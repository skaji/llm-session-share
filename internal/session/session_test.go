package session

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const testID = "01a0de0e-f326-7ec1-bc06-0aee7bbdf319"

func TestCodexOpenPageContext(t *testing.T) {
	const context = `<external_codex_apps_open_page>{"page_id":null}</external_codex_apps_open_page>`
	for _, tc := range []struct {
		name  string
		texts []string
		kinds []string
		want  string
	}{
		{"metadata", []string{context}, []string{"additional_content.codex_apps_open_page"}, ""},
		{"mixed content", []string{context, "question"}, []string{"additional_content.codex_apps_open_page", "user.text"}, "question"},
		{"legacy", []string{context}, nil, ""},
		{"legacy page ID", []string{`<external_codex_apps_open_page>{"page_id":"page-123"}</external_codex_apps_open_page>`}, nil, ""},
		{"legacy mixed content", []string{context, "question"}, nil, "question"},
		{"mismatched metadata", []string{context, "question"}, []string{"user.text"}, "question"},
		{"explicit user content", []string{context}, []string{"user.text"}, context},
		{"discussion", []string{"Please exclude " + context}, nil, "Please exclude " + context},
		{"following conversation", []string{context + "\nquestion"}, nil, context + "\nquestion"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := make([]map[string]string, 0, len(tc.texts))
			for _, text := range tc.texts {
				content = append(content, map[string]string{"type": "input_text", "text": text})
			}
			payload := map[string]any{"role": "user", "content": content}
			if tc.kinds != nil {
				payload["internal_chat_message_metadata_passthrough"] = map[string]any{"content_item_kinds": tc.kinds}
			}
			line, err := json.Marshal(map[string]any{"type": "response_item", "payload": payload})
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := Read(bytes.NewReader(line), testID+".jsonl", "skaji", "auto")
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if len(snapshot.Messages) != 0 {
					t.Fatalf("unexpected messages: %+v", snapshot.Messages)
				}
			} else if len(snapshot.Messages) != 1 || snapshot.Messages[0].Text != tc.want {
				t.Fatalf("messages=%+v; want %q", snapshot.Messages, tc.want)
			}
		})
	}
}

func TestReadZstandard(t *testing.T) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = encoder.Close() }()
	for _, data := range []string{
		`{"type":"session_meta","payload":{"id":"` + testID + `"}}` + "\n" + `{"role":"user","content":"Question"}` + "\n" + `{"role":"assistant","content":"Answer"}`,
		`{"type":"user","sessionId":"` + testID + `","message":{"role":"user","content":"Question"}}` + "\n" + `{"type":"assistant","message":{"role":"assistant","content":"Answer"}}`,
	} {
		want, err := Read(strings.NewReader(data), testID+".jsonl", "skaji", "auto")
		if err != nil {
			t.Fatal(err)
		}
		compressed := encoder.EncodeAll([]byte(data), nil)
		for _, suffix := range []string{".jsonl.zst", ".JSONL.ZST"} {
			got, err := Read(bytes.NewReader(compressed), testID+suffix, "skaji", "auto")
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: got %+v, %v; want %+v", suffix, got, err, want)
			}
		}
		for _, invalid := range [][]byte{[]byte("invalid zstd"), compressed[:len(compressed)-1]} {
			if _, err := Read(bytes.NewReader(invalid), testID+".jsonl.zst", "skaji", "auto"); err == nil {
				t.Fatal("expected error for invalid compressed data")
			}
		}
	}
}

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
