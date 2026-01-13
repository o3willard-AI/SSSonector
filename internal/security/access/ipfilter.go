package access

import (
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// IPFilterRule represents an IP filtering rule
type IPFilterRule struct {
	IPNet       *net.IPNet
	Action      string // "allow" or "deny"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   *time.Time
	Source      string // "manual", "automatic", etc.
}

// IPFilterManager manages IP filtering rules
type IPFilterManager struct {
	rules    []*IPFilterRule
	ruleLock sync.RWMutex
	logger   *zap.Logger
}

// NewIPFilterManager creates a new IP filter manager
func NewIPFilterManager(logger *zap.Logger) *IPFilterManager {
	return &IPFilterManager{
		rules:  make([]*IPFilterRule, 0),
		logger: logger,
	}
}

// AddRule adds a new IP filtering rule
func (m *IPFilterManager) AddRule(cidr string, action string, description string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %v", err)
	}

	if action != "allow" && action != "deny" {
		return fmt.Errorf("invalid action: %s (must be 'allow' or 'deny')", action)
	}

	rule := &IPFilterRule{
		IPNet:       ipNet,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Source:      "manual",
	}

	m.ruleLock.Lock()
	defer m.ruleLock.Unlock()

	m.rules = append(m.rules, rule)

	m.logger.Info("Added IP filter rule",
		zap.String("cidr", cidr),
		zap.String("action", action),
		zap.String("description", description))

	return nil
}

// AddTemporaryRule adds a temporary IP filtering rule with expiration
func (m *IPFilterManager) AddTemporaryRule(cidr string, action string, description string, duration time.Duration) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %v", err)
	}

	if action != "allow" && action != "deny" {
		return fmt.Errorf("invalid action: %s (must be 'allow' or 'deny')", action)
	}

	expiresAt := time.Now().Add(duration)

	rule := &IPFilterRule{
		IPNet:       ipNet,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   &expiresAt,
		Source:      "temporary",
	}

	m.ruleLock.Lock()
	defer m.ruleLock.Unlock()

	m.rules = append(m.rules, rule)

	m.logger.Info("Added temporary IP filter rule",
		zap.String("cidr", cidr),
		zap.String("action", action),
		zap.String("description", description),
		zap.Duration("duration", duration))

	return nil
}

// RemoveRule removes an IP filtering rule
func (m *IPFilterManager) RemoveRule(cidr string) error {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %v", err)
	}

	m.ruleLock.Lock()
	defer m.ruleLock.Unlock()

	newRules := []*IPFilterRule{}
	found := false

	for _, rule := range m.rules {
		if rule.IPNet.String() == ipNet.String() {
			found = true
			continue
		}
		newRules = append(newRules, rule)
	}

	if !found {
		return fmt.Errorf("rule for CIDR %s not found", cidr)
	}

	m.rules = newRules

	m.logger.Info("Removed IP filter rule",
		zap.String("cidr", cidr))

	return nil
}

// CheckIP checks if an IP address is allowed based on filtering rules
func (m *IPFilterManager) CheckIP(ipStr string) (bool, string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	m.ruleLock.RLock()
	defer m.ruleLock.RUnlock()

	// Clean up expired rules
	m.cleanupExpiredRules()

	// Check rules in order, first match wins
	for _, rule := range m.rules {
		if rule.IPNet.Contains(ip) {
			if rule.ExpiresAt != nil && time.Now().After(*rule.ExpiresAt) {
				continue // Skip expired rules
			}
			return rule.Action == "allow", rule.Description, nil
		}
	}

	// No matching rules - default to allow
	return true, "no matching rules", nil
}

// ListRules returns all IP filtering rules
func (m *IPFilterManager) ListRules() ([]*IPFilterRule, error) {
	m.ruleLock.RLock()
	defer m.ruleLock.RUnlock()

	// Clean up expired rules
	m.cleanupExpiredRules()

	// Return a copy of the rules
	rules := make([]*IPFilterRule, len(m.rules))
	copy(rules, m.rules)

	return rules, nil
}

// ClearExpiredRules removes all expired rules
func (m *IPFilterManager) ClearExpiredRules() {
	m.ruleLock.Lock()
	defer m.ruleLock.Unlock()

	m.cleanupExpiredRules()
}

// cleanupExpiredRules removes expired rules (internal method)
func (m *IPFilterManager) cleanupExpiredRules() {
	now := time.Now()
	newRules := []*IPFilterRule{}

	for _, rule := range m.rules {
		if rule.ExpiresAt == nil || now.Before(*rule.ExpiresAt) {
			newRules = append(newRules, rule)
		} else {
			m.logger.Info("Removed expired IP filter rule",
				zap.String("cidr", rule.IPNet.String()),
				zap.String("action", rule.Action))
		}
	}

	m.rules = newRules
}

// AddCommonSecurityRules adds common security filtering rules
func (m *IPFilterManager) AddCommonSecurityRules() error {
	// Allow private networks by default
	privateNetworks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
	}

	for _, cidr := range privateNetworks {
		if err := m.AddRule(cidr, "allow", "Private network range"); err != nil {
			return fmt.Errorf("failed to add private network rule for %s: %v", cidr, err)
		}
	}

	// Deny known malicious networks (examples)
	maliciousNetworks := []string{
		"1.2.3.0/24",   // Example malicious network
		"5.6.7.0/24",   // Example malicious network
		"9.10.11.0/24", // Example malicious network
	}

	for _, cidr := range maliciousNetworks {
		if err := m.AddRule(cidr, "deny", "Known malicious network"); err != nil {
			return fmt.Errorf("failed to add malicious network rule for %s: %v", cidr, err)
		}
	}

	m.logger.Info("Added common security IP filter rules",
		zap.Int("private_networks", len(privateNetworks)),
		zap.Int("malicious_networks", len(maliciousNetworks)))

	return nil
}

// AddRateLimitRule adds a temporary IP filter rule for rate limiting
func (m *IPFilterManager) AddRateLimitRule(ipStr string, duration time.Duration) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Create a /32 CIDR for the specific IP
	cidr := fmt.Sprintf("%s/32", ipStr)

	return m.AddTemporaryRule(cidr, "deny", "Rate limit exceeded", duration)
}

// CheckAndUpdateRateLimit checks IP and updates rate limit status
func (m *IPFilterManager) CheckAndUpdateRateLimit(ipStr string, maxAttempts int, attempt int) (bool, error) {
	allowed, description, err := m.CheckIP(ipStr)
	if err != nil {
		return false, err
	}

	if !allowed {
		m.logger.Debug("IP blocked by existing rule",
			zap.String("ip", ipStr),
			zap.String("reason", description))
		return false, nil
	}

	// If this is a rate limit attempt, add temporary block
	if attempt >= maxAttempts {
		if err := m.AddRateLimitRule(ipStr, 15*time.Minute); err != nil {
			return false, err
		}
		m.logger.Info("Rate limit exceeded, blocking IP",
			zap.String("ip", ipStr),
			zap.Duration("block_duration", 15*time.Minute))
		return false, nil
	}

	return true, nil
}
