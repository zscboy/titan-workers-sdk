package proxy

import (
	"net"
)

type Reqq struct {
	cap      int
	requests []*Request
	// lock      sync.Mutex
	freeIdx   *FreeIdx
	freeCount int
}

// this.cap = cap;
// this.requests = [];
// this.freeIdx = [];
// this.freeCount = cap;
// for (let i = 0; i < cap; i++) {
// 	let req = new reqbuilder(i);
// 	this.requests.push(req);
// 	this.freeIdx.push(i);
// }

// console.log("Reqq construct, cap:", cap);
func newReqq(cap int) *Reqq {
	requests := make([]*Request, 0, cap)
	freeIdx := newFreeIdx()
	for i := 0; i < cap; i++ {
		requests = append(requests, &Request{idx: uint16(i)})
		freeIdx.push(uint16(i))
	}

	return &Reqq{cap: cap, requests: requests, freeIdx: freeIdx, freeCount: cap}
}

func (r *Reqq) reqValid(idx, tag uint16) bool {
	if idx < 0 || idx >= uint16(len(r.requests)) {
		return false
	}

	req := r.requests[idx]
	if req.tag != tag {
		return false
	}

	return true
}

func (r *Reqq) getReq(idx, tag uint16) *Request {
	if idx < 0 || int(idx) > len(r.requests) {
		return nil
	}

	req := r.requests[idx]
	if req.tag != tag {
		return nil
	}

	return req
}

func (r *Reqq) allocReq(conn net.Conn) *Request {
	if r.freeIdx.length() < 1 {
		return nil
	}

	idx := r.freeIdx.pop()
	req := r.requests[idx]
	req.tag = req.tag + 1
	req.inused = true
	req.conn = conn

	r.freeCount--
	return req
}

func (r *Reqq) free(idx uint16, tag uint16) {
	length := len(r.requests)
	if idx < 0 || idx > uint16(length) {
		return
	}

	req := r.requests[idx]
	if req.tag != tag {
		return
	}

	req.dofree()

	req.inused = false
	req.tag = req.tag + 1

	r.freeIdx.push(idx)
	r.freeCount++
}

func (r *Reqq) isFulled() bool {
	return r.freeCount < 1
}

func (r *Reqq) reqCount() int {
	return r.cap - r.freeCount
}

func (r *Reqq) cleanup() {
	for _, request := range r.requests {
		if request.inused {
			r.free(request.idx, request.tag)
		}
	}
}
