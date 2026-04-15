package main

import "strings"

// findUserByEmail looks up a Jellyfin user ID and their stored EmailAddress by email.
// Returns ("", EmailAddress{}, false) if no match is found.
func (app *appContext) findUserByEmail(addr string) (string, EmailAddress, bool) {
	for _, em := range app.storage.GetEmails() {
		if strings.EqualFold(em.Addr, addr) {
			return em.JellyfinID, em, true
		}
	}
	return "", EmailAddress{}, false
}
