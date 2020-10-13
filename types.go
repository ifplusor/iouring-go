// +build linux

package iouring

import (
	"sync/atomic"

	iouring_syscall "github.com/iceber/iouring-go/syscall"
)

const (
	IOURING_FOUND_REQUEST          = 1
	IOURING_REQUEST_MAYBE_CANCELED = 2
)

var _zero uintptr

type SubmissionQueue struct {
	ptr  uintptr
	size uint32

	head    *uint32
	tail    *uint32
	mask    *uint32
	entries *uint32
	flags   *uint32
	dropped *uint32

	array []uint32
	sqes  []iouring_syscall.SubmissionQueueEntry

	sqeHead uint32
	sqeTail uint32
}

func (queue *SubmissionQueue) GetSQEntry() *iouring_syscall.SubmissionQueueEntry {
	head := atomic.LoadUint32(queue.head)
	next := queue.sqeTail + 1

	if (next - head) <= *queue.entries {
		sqe := &queue.sqes[queue.sqeTail&*queue.mask]
		queue.sqeTail = next
		sqe.Reset()
		return sqe
	}
	return nil
}

func (queue *SubmissionQueue) fallback(i uint32) {
	queue.sqeTail -= i
}

func (queue *SubmissionQueue) cqOverflow() bool {
	return (atomic.LoadUint32(queue.flags) & iouring_syscall.IORING_SQ_CQ_OVERFLOW) != 0
}

func (queue *SubmissionQueue) needWakeup() bool {
	return (atomic.LoadUint32(queue.flags) & iouring_syscall.IORING_SQ_NEED_WAKEUP) != 0
}

// sync internal status with kernel ring state on the SQ side
// return the number of pending items in the SQ ring, for the shared ring.
func (queue *SubmissionQueue) flush() int {
	if queue.sqeHead == queue.sqeTail {
		return int(*queue.tail - *queue.head)
	}

	tail := *queue.tail
	for toSubmit := queue.sqeTail - queue.sqeHead; toSubmit > 0; toSubmit-- {
		queue.array[tail&*queue.mask] = queue.sqeHead & *queue.mask
		tail++
		queue.sqeHead++
	}

	atomic.StoreUint32(queue.tail, tail)
	return int(tail - *queue.head)
}

type CompletionQueue struct {
	ptr  uintptr
	size uint32

	head     *uint32
	tail     *uint32
	mask     *uint32
	overflow *uint32
	entries  *uint32
	flags    *uint32

	cqes []iouring_syscall.CompletionQueueEvent
}

func (queue *CompletionQueue) peek() (cqe *iouring_syscall.CompletionQueueEvent) {
	head := *queue.head
	if head != atomic.LoadUint32(queue.tail) {
		//	if head < atomic.LoadUint32(queue.tail) {
		cqe = &queue.cqes[head&*queue.mask]
	}
	return
}

func (queue *CompletionQueue) advance(num uint32) {
	if num != 0 {
		atomic.AddUint32(queue.head, num)
	}
}

const (
	IORING_OP_NOP uint8 = iota
	IORING_OP_READV
	IORING_OP_WRITEV
	IORING_OP_FSYNC
	IORING_OP_READ_FIXED
	IORING_OP_WRITE_FIXED
	IORING_OP_POLL_ADD
	IORING_OP_POLL_REMOVE
	IORING_OP_SYNC_FILE_RANGE
	IORING_OP_SENDMSG
	IORING_OP_RECVMSG
	IORING_OP_TIMEOUT
	IORING_OP_TIMEOUT_REMOVE
	IORING_OP_ACCEPT
	IORING_OP_ASYNC_CANCEL
	IORING_OP_LINK_TIMEOUT
	IORING_OP_CONNECT
	IORING_OP_FALLOCATE
	IORING_OP_OPENAT
	IORING_OP_CLOSE
	IORING_OP_FILES_UPDATE
	IORING_OP_STATX
	IORING_OP_READ
	IORING_OP_WRITE
	IORING_OP_FADVISE
	IORING_OP_MADVISE
	IORING_OP_SEND
	IORING_OP_RECV
	IORING_OP_OPENAT2
	IORING_OP_EPOLL_CTL
	IORING_OP_SPLICE
	IORING_OP_PROVIDE_BUFFERS
	IORING_OP_REMOVE_BUFFERS
	IORING_OP_TEE
	IORING_OP_SHUTDOWN
)