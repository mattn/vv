package mpd

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestCommandList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	go func() {
		ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_ok_begin\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "clear\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "add \"/foo/bar\"\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nOK\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := &CommandList{}
	cl.Clear()
	cl.Add("/foo/bar")
	if err := c.ExecCommandList(ctx, cl); err != nil {
		t.Errorf("CommandList got error %v; want nil", err)
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}

func TestCommandListCommandError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ts, _ := mpdtest.NewServer("OK MPD 0.19")
	go func() {
		ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_ok_begin\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "clear\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "play 0\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "add \"/foo/bar\"\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nACK [2@1] {} Bad song index\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := &CommandList{}
	cl.Clear()
	cl.Play(0)
	cl.Add("/foo/bar")
	if err, want := c.ExecCommandList(ctx, cl), newCommandError("ACK [2@1] {} Bad song index"); !reflect.DeepEqual(err, want) {
		t.Errorf("CommandList got error %v; want %v", err, want)
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}

func TestCommandListNetworkError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ts, _ := mpdtest.NewServer("OK MPD 0.19")
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_ok_begin\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "clear\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "play 0\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "add \"/foo/bar\"\n"})
		ts.Expect(ctx, &mpdtest.WR{Read: "command_list_end\n"})
		ts.Disconnect(ctx)
		ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := &CommandList{}
	cl.Clear()
	cl.Play(0)
	cl.Add("/foo/bar")
	if err := c.ExecCommandList(ctx, cl); err == nil {
		t.Error("CommandList got nil; want non nil error at network error")
	}
	wg.Wait()
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}
