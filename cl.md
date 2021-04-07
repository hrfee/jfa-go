This one isn't super big but includes some pretty important bug fixes.

Features:
* Link-only password resets: Instead of the user having to enter a code, a magic link is sent which will reset the password once clicked. The password reset PIN input sometimes seems broken on Jellyfin (at least for me) so this is kind of a workaround.
* Proper time handling in the web UI: Times displayed on the invites & accounts tab now match the language you're using (e.g MM/DD/YY for en-US and DD/MM/YY for everywhere else). 12/24 hour time can also be toggled in the language menu in the top left.

Fixes:
* Fix missing "Last Active" time on newer versions of Jellyfin (#69)
    * Time parser was rewritten so it should handle a lot more formats automatically now.
* Fix bug where user expiry would change/disappear after a restart (#77)
* Add missing attributes that weren't being stored in profiles (#76)

Other:
* Massively cleaned up build files 
