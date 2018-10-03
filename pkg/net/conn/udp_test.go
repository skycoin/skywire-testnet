package conn

import "testing"

func TestRtt_Less(t *testing.T) {
	rs := newRttSampler(4)
	t.Log(rs.push(5))
	t.Log(rs.push(5))
	t.Log(rs.push(5))
	t.Log(rs.push(5))
	t.Log(rs.push(5))
	t.Log(rs.push(5))
	t.Log(rs.push(6))
	t.Log(rs.push(7))
	t.Log(rs.push(8))
	t.Log(rs.push(9))
	t.Log(rs.push(10))
}
