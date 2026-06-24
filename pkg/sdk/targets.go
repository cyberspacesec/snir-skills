package sdk

import "github.com/cyberspacesec/snir-skills/pkg/scan"

// ExpandTargets expands bare hosts/IPs using the same scheme and port rules as scan workflows.
func ExpandTargets(targets []string, screenshotOpts *ScreenshotOptions) []string {
	runnerOpts := mergeWithScreenshotOptions(toRunnerOptions(DefaultClientOptions()), screenshotOpts)
	return scan.ExpandTargets(targets, &runnerOpts)
}

// ExpandTarget expands a single bare host/IP using the same scheme and port rules as scan workflows.
func ExpandTarget(target string, screenshotOpts *ScreenshotOptions) []string {
	return ExpandTargets([]string{target}, screenshotOpts)
}

// ExpandTargets expands bare hosts/IPs using this client's default options plus per-request overrides.
func (c *Client) ExpandTargets(targets []string, screenshotOpts *ScreenshotOptions) []string {
	runnerOpts := mergeWithScreenshotOptions(toRunnerOptions(c.opts), screenshotOpts)
	return scan.ExpandTargets(targets, &runnerOpts)
}

// ExpandTarget expands one bare host/IP using this client's default options plus per-request overrides.
func (c *Client) ExpandTarget(target string, screenshotOpts *ScreenshotOptions) []string {
	return c.ExpandTargets([]string{target}, screenshotOpts)
}
