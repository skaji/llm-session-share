package session

import (
	"strings"
	"testing"
)

const testID = "01a0de0e-f326-7ec1-bc06-0aee7bbdf319"

func TestCodexConversation(t *testing.T) {
	t.Parallel()
	data := `{"type":"session_meta","payload":{"id":"` + testID + `"}}
{"type":"response_item","payload":{"role":"developer","content":[{"text":"hidden instructions"}]}}
{"type":"response_item","payload":{"role":"user","content":[{"text":"injected instructions"},{"text":"日本語の質問"},{"text":"data:image/png;base64,AAAA"}],"internal_chat_message_metadata_passthrough":{"content_item_kinds":["agents_md.instructions","user.text","user.text"]}}}
{"type":"response_item","payload":{"role":"user","content":[{"text":"<environment_context>hidden environment</environment_context>"}]}}
{"type":"response_item","payload":{"role":"assistant","phase":"commentary","content":[{"text":"Working on it."}]}}
{"type":"event_msg","payload":{"type":"agent_message","message":"Working on it."}}
{"type":"response_item","payload":{"type":"function_call","arguments":"hidden tool"}}
{"type":"response_item","payload":{"type":"function_call_output","output":"hidden output"}}
{"type":"response_item","payload":{"type":"reasoning","summary":[{"text":"hidden reasoning"}]}}
{"timestamp":"2026-09-27T00:00:00Z","type":"response_item","payload":{"role":"assistant","phase":"final_answer","content":[{"text":"Done."}]}}
`
	snapshot, err := Read(strings.NewReader(data), "anything.jsonl", "skaji", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ID != testID || snapshot.Source != "codex" || snapshot.Name != "skaji" {
		t.Fatalf("unexpected identity: %+v", snapshot)
	}
	if len(snapshot.Messages) != 3 || snapshot.Messages[0].Text != "日本語の質問" || snapshot.Messages[1].Text != "Working on it." || snapshot.Messages[2].Text != "Done." {
		t.Fatalf("unexpected conversation: %+v", snapshot.Messages)
	}
	if snapshot.Messages[2].Timestamp != "2026-09-27T00:00:00Z" {
		t.Fatal("lost message timestamp")
	}
}

func TestClaudeConversation(t *testing.T) {
	t.Parallel()
	data := `{"type":"user","sessionId":"` + testID + `","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>hidden</system-reminder>\nQuestion"},{"type":"tool_result","content":"hidden tool result"}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"hidden thought"},{"type":"tool_use","input":{"text":"hidden command"}},{"type":"text","text":"Answer"}]}}
{"type":"user","message":{"role":"user","content":"<command-name>/clear</command-name>"}}
{"type":"user","message":{"role":"user","content":"<task-notification>hidden notification</task-notification>"}}
{"type":"progress","message":{"role":"assistant","content":"duplicate answer"}}
{"type":"user","message":{"role":"assistant","content":"mismatched role"}}
`
	snapshot, err := Read(strings.NewReader(data), testID+".jsonl", "friend", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Source != "claude" || len(snapshot.Messages) != 2 || snapshot.Messages[0].Text != "Question" || snapshot.Messages[1].Text != "Answer" {
		t.Fatalf("unexpected conversation: %+v", snapshot)
	}
}

func TestPartialRecordAndLongMessages(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("hello", 20000)
	first := `{"role":"user","content":"` + long + `"}` + "\n"
	partial := `{"role":"assistant","content":"part`
	path := "rollout-2026-09-27T00-00-00-" + testID + ".jsonl"
	snapshot, err := Read(strings.NewReader(first+partial), path, "skaji", "auto")
	if err != nil || len(snapshot.Messages) != 1 || snapshot.Messages[0].Text != long {
		t.Fatalf("incomplete record or long message handling failed: %v", err)
	}
	snapshot, err = Read(strings.NewReader(first+partial+`ial"}`), path, "skaji", "auto")
	if err != nil || len(snapshot.Messages) != 2 || snapshot.Messages[1].Text != "partial" {
		t.Fatalf("completed final record was not read: %+v, %v", snapshot.Messages, err)
	}
}

func TestClaudeTitles(t *testing.T) {
	const path = "01a0de4a-2245-7102-b715-58d2ab09ea8f.jsonl"
	text := ""
	for _, item := range []struct{ record, want string }{
		{`{"type":"ai-title","aiTitle":"Generated"}`, "Generated"},
		{`{"type":"ai-title","aiTitle":"Later generated"}`, "Generated"},
		{`{"type":"custom-title","customTitle":" Custom "}`, "Custom"},
		{`{"type":"ai-title","aiTitle":"Ignored"}`, "Custom"},
		{`{"type":"custom-title","customTitle":"Renamed"}`, "Renamed"},
		{`{"type":"custom-title","customTitle":"  "}`, "Renamed"},
	} {
		text += item.record + "\n"
		snapshot, err := Read(strings.NewReader(text), path, "skaji", "auto")
		if err != nil || snapshot.Title != item.want || len(snapshot.Messages) != 0 {
			t.Fatalf("got %+v, %v; want %q", snapshot, err, item.want)
		}
	}
}
