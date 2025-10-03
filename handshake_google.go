package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"
)

// HandshakeMessage is the structure of the message we will send via Google Pub/Sub.
type HandshakeMessage struct {
	PublicKey  []byte `json:"public_key"`
	ReplyTopic string `json:"reply_topic"`
	// The client will send its real IP:Port for the server to connect back to.
	RealIP string `json:"real_ip"`
}

// performGoogleHandshakeServer is the "listener" side of the handshake.
// IT NOW CORRECTLY RETURNS 3 VALUES.
func performGoogleHandshakeServer(projectID, topicID string) (*[KeySize]byte, string, error) {
	fmt.Println("?? Starting Google Pub/Sub handshake (Listen Mode)...")
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentialsFile("gcloud-key.json"))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create pubsub client: %w", err)
	}
	defer client.Close()

	topic := client.Topic(topicID)
	subID := fmt.Sprintf("%s-sub-%d", topicID, time.Now().UnixNano())
	sub, err := client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{Topic: topic})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create subscription: %w", err)
	}
	defer sub.Delete(ctx)

	fmt.Printf("Listening for handshake on topic '%s'...\n", topicID)

	var clientMsg HandshakeMessage
	cctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	err = sub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		if err := json.Unmarshal(msg.Data, &clientMsg); err != nil {
			log.Printf("Could not decode handshake message: %v", err)
			return
		}
		cancel()
	})
	if err != nil && err != context.Canceled {
		return nil, "", fmt.Errorf("failed to receive handshake message: %w", err)
	}
    if clientMsg.PublicKey == nil {
        return nil, "", fmt.Errorf("did not receive a valid handshake message in time")
    }

	fmt.Println("Received client handshake message.")

	clientPubKey, err := curve.NewPublicKey(clientMsg.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("invalid client public key: %v", err)
	}

	serverPrivKey, err := GenerateKeys()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate server keys: %w", err)
	}

	replyTopic := client.Topic(clientMsg.ReplyTopic)
	replyMsg, _ := json.Marshal(HandshakeMessage{PublicKey: serverPrivKey.PublicKey().Bytes()})
	result := replyTopic.Publish(ctx, &pubsub.Message{Data: replyMsg})
	if _, err := result.Get(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to publish reply: %w", err)
	}

	fmt.Println("Sent handshake reply.")

	sessionKey, err := CalculateSharedSecret(serverPrivKey, clientPubKey)
	if err != nil {
		return nil, "", err
	}

	return sessionKey, clientMsg.RealIP, nil
}

// performGoogleHandshakeClient is the "initiator" side of the handshake.
// IT NOW CORRECTLY ACCEPTS 4 ARGUMENTS.
func performGoogleHandshakeClient(projectID, topicID, connectAddr string) (*[KeySize]byte, error) {
	fmt.Println("?? Starting Google Pub/Sub handshake (Connect Mode)...")
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentialsFile("gcloud-key.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}
	defer client.Close()

	clientPrivKey, err := GenerateKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client keys: %w", err)
	}

	replyTopicID := fmt.Sprintf("reply-%d", time.Now().UnixNano())
	replyTopic, err := client.CreateTopic(ctx, replyTopicID)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply topic: %w", err)
	}
	defer replyTopic.Delete(ctx)

	replySub, err := client.CreateSubscription(ctx, replyTopicID+"-sub", pubsub.SubscriptionConfig{Topic: replyTopic})
	if err != nil {
		return nil, fmt.Errorf("failed to create reply subscription: %w", err)
	}
	defer replySub.Delete(ctx)

	mainTopic := client.Topic(topicID)
	msg, _ := json.Marshal(HandshakeMessage{
		PublicKey:  clientPrivKey.PublicKey().Bytes(),
		ReplyTopic: replyTopicID,
		RealIP:     connectAddr,
	})
	result := mainTopic.Publish(ctx, &pubsub.Message{Data: msg})
	if _, err := result.Get(ctx); err != nil {
		return nil, fmt.Errorf("failed to publish handshake: %w", err)
	}
	fmt.Println("Sent handshake message, waiting for reply...")

	var serverMsg HandshakeMessage
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = replySub.Receive(cctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		json.Unmarshal(msg.Data, &serverMsg)
		cancel()
	})
	if err != nil && err != context.Canceled {
		return nil, fmt.Errorf("failed to receive reply: %w", err)
	}
    if serverMsg.PublicKey == nil {
        return nil, fmt.Errorf("did not receive a valid server reply in time")
    }

	fmt.Println("Received server handshake reply.")

	serverPubKey, err := curve.NewPublicKey(serverMsg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid server public key: %v", err)
	}

	return CalculateSharedSecret(clientPrivKey, serverPubKey)
}