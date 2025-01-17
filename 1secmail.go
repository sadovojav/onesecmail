package onesecmail

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
)

const apiBase = "https://www.1secmail.com/api/v1/"

type mailboxAction int

const (
	getMessages mailboxAction = iota
	readMessage
	download
	getDomainList
)

func (m mailboxAction) String() string {
	return [...]string{"getMessages", "readMessage", "download", "getDomainList"}[m]
}

// Mail represents a mail in a 1secmail inbox.
type Mail struct {
	ID          int          `json:"id"`
	From        string       `json:"from"`
	Subject     string       `json:"subject"`
	Date        string       `json:"date"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Body        *string      `json:"body,omitempty"`
	TextBody    *string      `json:"textBody,omitempty"`
	HTMLBody    *string      `json:"htmlBody,omitempty"`
}

// Attachment represents an attachment in a 1secmail mail.
type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
}

// HTTPClient is an interface that makes an HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Mailbox manages communication with 1secmail's API.
type Mailbox struct {
	Login   string
	Domain  string
	BaseURL string // Base URL for API requests, with trailing slash.
	client  HTTPClient
}

// Address returns the email address of a Mailbox.
func (m Mailbox) Address() string {
	return fmt.Sprintf("%s@%s", m.Login, m.Domain)
}

// NewMailbox returns a new Mailbox. Use login and domain for the email
// handler that you intend to use. Login is the email username.
// If nil httpClient is provided, a new http.Client will be created.
func NewMailbox(login, domain string, httpClient HTTPClient) *Mailbox {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	mail := &Mailbox{
		BaseURL: apiBase,
		client:  httpClient,
		Login:   login,
	}

	domains, err := mail.GetDomainList()
	if err != nil {
		log.Panic(err)
	}

	if domain == "" {
		mail.Domain = domains[rand.Intn(len(domains))]
	} else {
		if !contains(domains, domain) {
			log.Panic("does not valid domain")
		}

		mail.Domain = domain
	}

	return mail
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// CheckInbox checks the inbox of a mailbox, and returns a list of mails.
func (m Mailbox) CheckInbox() ([]*Mail, error) {
	req := constructRequest("GET", m.BaseURL, getMessages, map[string]string{
		"login":  m.Login,
		"domain": m.Domain,
	})
	resp, err := m.client.Do(req)
	if err != nil || (resp != nil && resp.StatusCode != 200) {
		return nil, fmt.Errorf("check inbox failed: %w, error code: %v", err, resp.StatusCode)
	}
	defer resp.Body.Close()

	var mails []*Mail
	if err := json.NewDecoder(resp.Body).Decode(&mails); err != nil {
		return nil, fmt.Errorf("decode JSON failed: %w", err)
	}
	return mails, nil
}

// ReadMessage retrieves a particular mail from the inbox of a mailbox.
func (m Mailbox) ReadMessage(messageID int) (*Mail, error) {
	req := constructRequest("GET", m.BaseURL, readMessage, map[string]string{
		"login":  m.Login,
		"domain": m.Domain,
		"id":     fmt.Sprint(messageID),
	})
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read messsage failed: %w", err)
	}
	defer resp.Body.Close()

	var mail *Mail
	if err := json.NewDecoder(resp.Body).Decode(&mail); err != nil {
		return nil, fmt.Errorf("decode JSON failed: %w", err)
	}

	return mail, nil
}

// GetDomainList retrieves a list of currently active handling domains
func (m Mailbox) GetDomainList() ([]string, error) {
	req := constructRequest("GET", m.BaseURL, getDomainList, map[string]string{})
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read messsage failed: %w", err)
	}
	defer resp.Body.Close()

	var domains []string
	if err := json.NewDecoder(resp.Body).Decode(&domains); err != nil {
		return nil, fmt.Errorf("decode JSON failed: %w", err)
	}

	return domains, nil
}

func constructRequest(method, baseURL string, action mailboxAction, args map[string]string) *http.Request {
	req, _ := http.NewRequest(method, baseURL, nil)
	query := req.URL.Query()
	query.Add("action", fmt.Sprint(action))
	for k, v := range args {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()
	return req
}
