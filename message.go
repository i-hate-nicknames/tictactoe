package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MSG_QUESTION = "QUE"
	MSG_ANSWER   = "ANS"
	SEPARATOR    = "|"
)

type QuestionMessage struct {
	Text string
}

type AnswerMessage struct {
	Text   string
	Advice string
}

func MarshalMessage(message interface{}) (string, error) {
	var msgType string
	switch message.(type) {
	case QuestionMessage:
		msgType = MSG_QUESTION
	case AnswerMessage:
		msgType = MSG_ANSWER
	default:
		return "", fmt.Errorf("%v of type %T is not a valid message to marshal", message, message)
	}
	payload, err := json.Marshal(message)
	if err != nil {
		return "", nil
	}
	return msgType + SEPARATOR + string(payload), nil
}

func UnmarshalMessage(marshalled string) (interface{}, error) {
	parts := strings.Split(marshalled, SEPARATOR)
	if len(parts) != 2 {
		return nil, fmt.Errorf("Unmarshal: malformed message: %s", marshalled)
	}
	msgType, payload := parts[0], []byte(parts[1])
	switch msgType {
	case MSG_QUESTION:
		var message QuestionMessage
		err := json.Unmarshal(payload, &message)
		if err != nil {
			return nil, err
		}
		return message, nil
	case MSG_ANSWER:
		var message AnswerMessage
		err := json.Unmarshal(payload, &message)
		if err != nil {
			return nil, err
		}
		return message, nil
	default:
		return nil, fmt.Errorf("Unrecognized message type %s", msgType)
	}
}
