package apicompat

import "testing"

// TestAnthropicEventToResponses_TextEmitsContentPart pins that a message text
// stream emits response.content_part.added, and that it precedes the first
// output_text.delta for that part.
//
// Why: the OpenAI SDK's accumulating stream helper (client.responses.stream)
// only appends a content part to the message item when it sees
// content_part.added. The item is added with content: [], so a missing event
// makes the following output_text.delta index output.content[content_index] and
// raise IndexError. Raw event iteration does not accumulate, so a regression
// here is easy to miss.
func TestAnthropicEventToResponses_TextEmitsContentPart(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var types []string
	feed := func(evt *AnthropicStreamEvent) {
		for _, out := range AnthropicEventToResponsesEvents(evt, state) {
			types = append(types, out.Type)
		}
	}

	idx := 0
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1", Model: "claude-sonnet-4-5"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &idx, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{Type: "text_delta", Text: "Hel"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{Type: "text_delta", Text: "lo"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &idx})
	feed(&AnthropicStreamEvent{Type: "message_stop"})

	posOf := func(target string) int {
		for i, ty := range types {
			if ty == target {
				return i
			}
		}
		return -1
	}

	partAdded := posOf("response.content_part.added")
	firstDelta := posOf("response.output_text.delta")

	if partAdded < 0 {
		t.Fatalf("response.content_part.added was not emitted; got %v", types)
	}
	if firstDelta < 0 {
		t.Fatalf("response.output_text.delta was not emitted; got %v", types)
	}
	if partAdded > firstDelta {
		t.Errorf("content_part.added must precede the first output_text.delta; got %v", types)
	}
	if posOf("response.content_part.done") < 0 {
		t.Errorf("response.content_part.done was not emitted; got %v", types)
	}
}

// TestAnthropicEventToResponses_DoneEventsCarryFullText pins that done events
// carry the part's full text (deltas carry increments only).
func TestAnthropicEventToResponses_DoneEventsCarryFullText(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var events []ResponsesStreamEvent
	feed := func(evt *AnthropicStreamEvent) {
		events = append(events, AnthropicEventToResponsesEvents(evt, state)...)
	}

	idx := 0
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &idx, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{Type: "text_delta", Text: "Hello "}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{Type: "text_delta", Text: "world"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &idx})

	const want = "Hello world"
	var sawTextDone, sawPartDone bool
	for _, e := range events {
		switch e.Type {
		case "response.output_text.done":
			sawTextDone = true
			if e.Text != want {
				t.Errorf("output_text.done text = %q, want %q", e.Text, want)
			}
		case "response.content_part.done":
			sawPartDone = true
			if e.Part == nil || e.Part.Text != want {
				t.Errorf("content_part.done part = %+v, want text %q", e.Part, want)
			}
		}
	}
	if !sawTextDone || !sawPartDone {
		t.Errorf("missing done events: output_text.done=%v content_part.done=%v", sawTextDone, sawPartDone)
	}
}

func TestAnthropicEventToResponses_ConsecutiveTextBlocksUseDistinctContentIndexes(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var events []ResponsesStreamEvent
	feed := func(evt *AnthropicStreamEvent) {
		events = append(events, AnthropicEventToResponsesEvents(evt, state)...)
	}

	firstIndex, secondIndex := 0, 1
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &firstIndex, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &firstIndex, Delta: &AnthropicDelta{Type: "text_delta", Text: "first"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &firstIndex})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &secondIndex, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &secondIndex, Delta: &AnthropicDelta{Type: "text_delta", Text: "second"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &secondIndex})
	feed(&AnthropicStreamEvent{Type: "message_stop"})

	wantContentIndexes := []int{0, 1}
	for _, eventType := range []string{
		"response.content_part.added",
		"response.output_text.delta",
		"response.output_text.done",
		"response.content_part.done",
	} {
		var got []int
		var itemIDs []string
		for _, event := range events {
			if event.Type == eventType {
				got = append(got, event.ContentIndex)
				itemIDs = append(itemIDs, event.ItemID)
			}
		}
		if len(got) != len(wantContentIndexes) || got[0] != wantContentIndexes[0] || got[1] != wantContentIndexes[1] {
			t.Errorf("%s content indexes = %v, want %v", eventType, got, wantContentIndexes)
		}
		if len(itemIDs) == 2 && itemIDs[0] != itemIDs[1] {
			t.Errorf("%s item IDs = %v, consecutive text blocks must share one message item", eventType, itemIDs)
		}
	}

	var outputItemAdded, outputItemDone int
	var completed *ResponsesStreamEvent
	for i := range events {
		switch events[i].Type {
		case "response.output_item.added":
			outputItemAdded++
		case "response.output_item.done":
			outputItemDone++
		case "response.completed":
			completed = &events[i]
		}
	}
	if outputItemAdded != 1 || outputItemDone != 1 {
		t.Errorf("message item lifecycle: added=%d done=%d, want one shared item", outputItemAdded, outputItemDone)
	}
	if completed == nil || completed.Response == nil || len(completed.Response.Output) != 1 {
		t.Fatalf("completed response output = %+v, want one message", completed)
	}
	content := completed.Response.Output[0].Content
	if len(content) != 2 || content[0].Text != "first" || content[1].Text != "second" {
		t.Fatalf("completed message content = %+v, want first and second text parts", content)
	}
}

func TestAnthropicEventToResponses_TextThenThinkingClosesMessageBeforeReasoning(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var events []ResponsesStreamEvent
	feed := func(evt *AnthropicStreamEvent) {
		events = append(events, AnthropicEventToResponsesEvents(evt, state)...)
	}

	textIndex, thinkingIndex := 0, 1
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &textIndex, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &textIndex, Delta: &AnthropicDelta{Type: "text_delta", Text: "answer"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &textIndex})
	beforeThinkingStart := len(events)
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &thinkingIndex, ContentBlock: &AnthropicContentBlock{Type: "thinking"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &thinkingIndex, Delta: &AnthropicDelta{Type: "thinking_delta", Thinking: "reason"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &thinkingIndex})
	feed(&AnthropicStreamEvent{Type: "message_stop"})

	transition := events[beforeThinkingStart:]
	if len(transition) < 2 || transition[0].Type != "response.output_item.done" || transition[1].Type != "response.output_item.added" {
		t.Fatalf("text-to-thinking transition starts with %v, want message done then reasoning added", eventTypes(transition))
	}
	if transition[0].Item == nil || transition[0].Item.Type != "message" || transition[0].OutputIndex != 0 {
		t.Fatalf("closed item = %+v, want message at output index 0", transition[0])
	}
	if transition[1].Item == nil || transition[1].Item.Type != "reasoning" || transition[1].OutputIndex != 1 {
		t.Fatalf("opened item = %+v, want reasoning at output index 1", transition[1])
	}

	var completed *ResponsesStreamEvent
	for i := range events {
		if events[i].Type == "response.completed" {
			completed = &events[i]
		}
	}
	if completed == nil || completed.Response == nil || len(completed.Response.Output) != 2 {
		t.Fatalf("completed response output = %+v, want message and reasoning", completed)
	}
	message, reasoning := completed.Response.Output[0], completed.Response.Output[1]
	if message.Type != "message" || len(message.Content) != 1 || message.Content[0].Text != "answer" {
		t.Errorf("first output = %+v, want preserved message", message)
	}
	if reasoning.Type != "reasoning" || len(reasoning.Summary) != 1 || reasoning.Summary[0].Text != "reason" {
		t.Errorf("second output = %+v, want preserved reasoning", reasoning)
	}
}

func eventTypes(events []ResponsesStreamEvent) []string {
	types := make([]string, len(events))
	for i := range events {
		types[i] = events[i].Type
	}
	return types
}

// TestAnthropicEventToResponses_CompletedCarriesOutput pins that
// response.completed carries the full output list. The SDK's
// get_final_response() and tracing integrations parse the terminal event's
// response directly; an empty output leaves them with nothing (the text still
// renders from deltas, which is why this is invisible when only watching the
// stream).
func TestAnthropicEventToResponses_CompletedCarriesOutput(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var events []ResponsesStreamEvent
	feed := func(evt *AnthropicStreamEvent) {
		events = append(events, AnthropicEventToResponsesEvents(evt, state)...)
	}

	idx := 0
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &idx, ContentBlock: &AnthropicContentBlock{Type: "text"}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{Type: "text_delta", Text: "4826"}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &idx})
	feed(&AnthropicStreamEvent{Type: "message_stop"})

	var completed *ResponsesStreamEvent
	for i := range events {
		if events[i].Type == "response.completed" {
			completed = &events[i]
		}
	}
	if completed == nil || completed.Response == nil {
		t.Fatalf("response.completed was not emitted")
	}
	if len(completed.Response.Output) == 0 {
		t.Fatalf("response.completed carries an empty output; clients would see no result")
	}
	msg := completed.Response.Output[0]
	if msg.Type != "message" || len(msg.Content) == 0 {
		t.Fatalf("output[0] = %+v, want a message with content", msg)
	}
	if msg.Content[0].Text != "4826" {
		t.Errorf("output[0].content[0].text = %q, want %q", msg.Content[0].Text, "4826")
	}
}

// TestAnthropicEventToResponses_ToolCallCompletedCarriesArguments pins that a
// function call's accumulated arguments survive into output_item.done and
// response.completed.
func TestAnthropicEventToResponses_ToolCallCompletedCarriesArguments(t *testing.T) {
	state := NewAnthropicEventToResponsesState()
	state.Model = "claude-sonnet-4-5"

	var events []ResponsesStreamEvent
	feed := func(evt *AnthropicStreamEvent) {
		events = append(events, AnthropicEventToResponsesEvents(evt, state)...)
	}

	idx := 0
	feed(&AnthropicStreamEvent{Type: "message_start", Message: &AnthropicResponse{ID: "msg_1"}})
	feed(&AnthropicStreamEvent{Type: "content_block_start", Index: &idx, ContentBlock: &AnthropicContentBlock{
		Type: "tool_use", ID: "toolu_1", Name: "get_weather",
	}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{
		Type: "input_json_delta", PartialJSON: `{"city":`,
	}})
	feed(&AnthropicStreamEvent{Type: "content_block_delta", Index: &idx, Delta: &AnthropicDelta{
		Type: "input_json_delta", PartialJSON: `"SH"}`,
	}})
	feed(&AnthropicStreamEvent{Type: "content_block_stop", Index: &idx})
	feed(&AnthropicStreamEvent{Type: "message_stop"})

	var completed *ResponsesStreamEvent
	for i := range events {
		if events[i].Type == "response.completed" {
			completed = &events[i]
		}
	}
	if completed == nil || completed.Response == nil || len(completed.Response.Output) == 0 {
		t.Fatalf("response.completed carries no output")
	}
	fc := completed.Response.Output[0]
	if fc.Type != "function_call" {
		t.Fatalf("output[0].type = %q, want function_call", fc.Type)
	}
	if fc.Arguments != `{"city":"SH"}` {
		t.Errorf("arguments = %q, want %q", fc.Arguments, `{"city":"SH"}`)
	}
	if fc.Name != "get_weather" {
		t.Errorf("name = %q, want get_weather", fc.Name)
	}
}
