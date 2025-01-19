package slim

import "slices"

// nodeTyp 节点类型
type nodeTyp uint8

const (
	ntStatic nodeTyp = iota // /home
	ntParam                 // /:user
	ntAny                   // /*param

	pathSeparator = '/'
	paramLabel    = ':'
	anyLabel      = '*'
)

// split 以 `/` 作为分隔字符分割路由表达式，自动剔除多余的分隔字符。
// 第一个返回值表示分割后的片段列表；第二个返回值表示是否以 `/` 结尾。
// 当表达式表示根路由（即参数 s 的值为 "/"）时，第一个返回值为零值，第二个参数为 true。
func split(s string) ([]string, bool) {
	if s == "" {
		s = "/"
	} else if s[0] != pathSeparator {
		s = "/" + s
	}
	var segments []string
	start := -1
	l := len(s)
	for i := 0; i < l; i++ {
		if s[i] != pathSeparator {
			continue
		}
		if start == -1 {
			start = i
			continue
		}
		if start+1 < i {
			segments = append(segments, s[start:i])
		}
		start = i
	}
	if start+1 == l {
		return segments, true
	}
	if start > -1 && start < l-1 {
		segments = append(segments, s[start:])
	}
	return segments, false
}

// leaf 节点叶子
type leaf struct {
	// endpoints 服务端点，对应不同的路由
	endpoints endpoints
	// paramsCount 参数数量
	paramsCount int
}

// endpoint 通过方法查找提供服务的端点
func (l *leaf) endpoint(method string) *endpoint {
	for _, e := range l.endpoints {
		if e.method == method {
			return e
		}
	}
	return nil
}

// match 匹配端点。
// 第一个返回值表示允许发起请求的方法；
// 第二个返回值表示提供服务的端点。
func (l *leaf) match(method string) ([]string, *endpoint) {
	var ep *endpoint
	ms := make([]string, len(l.endpoints))
	for i, e := range l.endpoints {
		if e.method == method {
			ep = e
		}
		ms[i] = e.method
	}
	return ms, ep
}

// endpoint 服务端点
type endpoint struct {
	method        string // 支持的请求方法
	pattern       string // 注册端点的路由表达式
	trailingSlash bool   // 是否斜线结尾
	routeId       uint32 // 路由编号
}

type endpoints []*endpoint

func (e endpoints) Len() int           { return len(e) }
func (e endpoints) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e endpoints) Less(i, j int) bool { return e[i].method < e[j].method }

// node 路由节点
// 基于路径分割符的前缀树
type node struct {
	// typ 节点类型
	typ nodeTyp
	// parent 上级节点
	parent *node
	// segment 节点表达式，为静态节点时有效
	segment string
	// leaf 节点叶子
	leaf *leaf
	// leafCount 叶子数量
	// 统计经由当前节点及其散发的子节点上的叶子节点数量，
	// 当数量归零时，表示这个节点及其子节点已经失去了提供
	// 服务的能力，此时，我们就可以删除它不会产生副作用。
	leafCount int
	// staticChildren 静态子节点
	staticChildren []*node
	// paramChild 参数子节点
	paramChild *node
	//  通配子节点
	anyChild *node
}

// insert 插入子节点
// 第一个返回值表示能够提供端点服务的节点；
// 第二个返回值表示是否新增叶子节点。
func (n *node) insert(segments []string, params *[]string, depth int) (tail *node, ok bool) {
	if depth == len(segments) {
		if n.leaf == nil {
			n.leaf = &leaf{paramsCount: len(*params)}
			n.leafCount++
			ok = true
		}
		tail = n
		return
	}

	var child *node
	var param string
	segment := segments[depth]
	switch segment[1] {
	case paramLabel:
		param = segment[2:]
		if n.paramChild == nil {
			child = &node{typ: ntParam, parent: n}
			n.paramChild = child
		} else {
			child = n.paramChild
		}
	case anyLabel:
		param = segment[2:]
		if param == "" {
			param = "*"
		}
		if n.anyChild == nil {
			child = &node{typ: ntAny, parent: n}
			n.anyChild = child
		} else {
			child = n.anyChild
		}
	default:
		for _, static := range n.staticChildren {
			if static.segment == segment {
				child = static
				break
			}
		}
		if child == nil {
			child = &node{typ: ntStatic, parent: n, segment: segment}
			n.staticChildren = append(n.staticChildren, child)
		}
	}
	if param != "" {
		*params = append(*params, segment[2:])
	}
	tail, ok = child.insert(segments, params, depth+1)
	if ok {
		n.leafCount++
	}
	return
}

// find 查找能够提供端点服务的节点
func (n *node) match(segments []string, depth int) *node {
	if len(segments) == depth {
		if n.leaf == nil {
			return nil
		}
		return n
	}
	segment := segments[depth]
	// 静态节点优先级最高
	for _, child := range n.staticChildren {
		if child.segment == segment {
			result := child.match(segments, depth+1)
			if result != nil {
				return result
			}
		}
	}
	// 其次是参数节点
	if n.paramChild != nil {
		result := n.paramChild.match(segments, depth+1)
		if result != nil {
			return result
		}
	}
	// 通配节点没有子节点，所以我们
	// 走到了终点站，直接返回它就行了。
	return n.anyChild
}

// remove 移除服务端点
// 如果参数 methods 为空，表示移除节点叶子上的所有服务端点。
// 第一个返回值是被移除端点关联的路由编号；
// 第二个返回值表示是否有端点被移除成功。
func (n *node) remove(
	methods []string,
	trailingSlash bool,
	routingTrailingSlash bool,
	segments []string,
	depth int,
) (routes []uint32, ok bool) {
	defer func() {
		if ok {
			n.leafCount--
			if n.leafCount <= 0 {
				n.staticChildren = nil
				n.paramChild = nil
				n.anyChild = nil
			}
		}
	}()

	if len(segments) == depth {
		// 只有叶子节点才提供端点服务
		if n.leaf == nil {
			return
		}
		if len(methods) > 0 {
			// 移除指定请求方法的服务端点
			for _, method := range methods {
				for i, e := range n.leaf.endpoints {
					if e.method == method && (routingTrailingSlash || e.trailingSlash == trailingSlash) {
						n.leaf.endpoints = slices.Delete(n.leaf.endpoints, i, 1)
						routes = append(routes, e.routeId)
						break
					}
				}
			}
			ok = len(routes) > 0
			if len(n.leaf.endpoints) == 0 {
				n.leaf = nil
			}
		} else {
			for _, e := range n.leaf.endpoints {
				routes = append(routes, e.routeId)
			}
			ok = true
			n.leaf = nil
		}
		return
	}

	segment := segments[depth]
	switch segment[1] {
	case paramLabel:
		if n.paramChild != nil {
			routes, ok = n.paramChild.remove(methods, trailingSlash, routingTrailingSlash, segments, depth+1)
			if n.leafCount <= 0 {
				n.paramChild = nil
			}
		}
	case anyLabel:
		if n.anyChild != nil {
			routes, ok = n.anyChild.remove(methods, trailingSlash, routingTrailingSlash, segments, depth+1)
			if n.anyChild.leafCount <= 0 {
				n.anyChild = nil
			}
		}
	default:
		for i, static := range n.staticChildren {
			if static.segment == segment {
				rms, yes := static.remove(methods, trailingSlash, routingTrailingSlash, segments, depth+1)
				if yes {
					routes, ok = rms, yes
					if static.leafCount <= 0 {
						n.staticChildren = slices.Delete(n.staticChildren, i, 1)
					}
				}
				return
			}
		}
	}
	return
}
