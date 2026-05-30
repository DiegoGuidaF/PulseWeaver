//go:build test

package collate_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/collate"
	"github.com/matryer/is"
)

type row struct {
	pid   int
	pname string
	cid   int // 0 = no child (e.g. a NULL LEFT JOIN)
	cname string
}

type parent struct {
	ID       int
	Name     string
	Children []string
}

func collapseRows(rows []row) []parent {
	return collate.Collapse(rows,
		func(r row) int { return r.pid },
		func(r row) parent { return parent{ID: r.pid, Name: r.pname, Children: []string{}} },
		func(r row) (string, bool) {
			if r.cid == 0 {
				return "", false
			}
			return r.cname, true
		},
		func(p *parent, c string) { p.Children = append(p.Children, c) },
	)
}

func TestCollapse_Empty(t *testing.T) {
	is := is.New(t)
	out := collapseRows(nil)
	is.Equal(len(out), 0)
	is.True(out != nil) // non-nil so it marshals to []
}

func TestCollapse_SingleParentNoChildren(t *testing.T) {
	is := is.New(t)
	out := collapseRows([]row{{pid: 1, pname: "a"}})
	is.Equal(len(out), 1)
	is.Equal(out[0].ID, 1)
	is.Equal(out[0].Name, "a")
	is.Equal(out[0].Children, []string{})
}

func TestCollapse_SingleParentMultipleChildren(t *testing.T) {
	is := is.New(t)
	out := collapseRows([]row{
		{pid: 1, pname: "a", cid: 10, cname: "x"},
		{pid: 1, pname: "a", cid: 11, cname: "y"},
	})
	is.Equal(len(out), 1)
	is.Equal(out[0].Children, []string{"x", "y"}) // attached in input order
}

func TestCollapse_InterleavedKeysPreserveFirstSeenOrder(t *testing.T) {
	is := is.New(t)
	out := collapseRows([]row{
		{pid: 2, pname: "b", cid: 20, cname: "p"},
		{pid: 1, pname: "a", cid: 10, cname: "x"},
		{pid: 2, pname: "b", cid: 21, cname: "q"},
		{pid: 1, pname: "a", cid: 11, cname: "y"},
	})
	is.Equal(len(out), 2)
	is.Equal(out[0].ID, 2) // first seen
	is.Equal(out[0].Children, []string{"p", "q"})
	is.Equal(out[1].ID, 1)
	is.Equal(out[1].Children, []string{"x", "y"})
}

func TestCollapse_SkipsNullChildren(t *testing.T) {
	is := is.New(t)
	out := collapseRows([]row{
		{pid: 1, pname: "a"},                      // no child
		{pid: 1, pname: "a", cid: 10, cname: "x"}, // one child
	})
	is.Equal(len(out), 1)
	is.Equal(out[0].Children, []string{"x"})
}

func TestGroupByMap(t *testing.T) {
	is := is.New(t)
	out := collate.GroupByMap([]row{
		{pid: 1, cname: "x"},
		{pid: 2, cname: "p"},
		{pid: 1, cname: "y"},
	},
		func(r row) int { return r.pid },
		func(r row) string { return r.cname },
	)
	is.Equal(len(out), 2)
	is.Equal(out[1], []string{"x", "y"}) // input order within group
	is.Equal(out[2], []string{"p"})
}

func TestGroupByMap_Empty(t *testing.T) {
	is := is.New(t)
	out := collate.GroupByMap(nil,
		func(r row) int { return r.pid },
		func(r row) string { return r.cname },
	)
	is.Equal(len(out), 0)
}

func TestOrEmpty(t *testing.T) {
	is := is.New(t)
	is.Equal(collate.OrEmpty[int](nil), []int{}) // nil → non-nil empty
	is.Equal(collate.OrEmpty([]int{1, 2}), []int{1, 2})
}
