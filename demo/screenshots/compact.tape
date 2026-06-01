Set Shell "bash"
Set Width 760
Set Height 840
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 12ms

Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Ubuntu Dev Instance Setup/
Down 11
Space
Down 8
Space
Down
Space
Enter
Wait+Screen /Timezone/
Enter
Wait+Screen /Target User/
Type "dev"
Enter
Wait+Screen /SSH public key/
Type "ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAA== setup-demo"
Enter
Wait+Screen /Confirm Run/
Enter
Wait+Screen /Provisioning complete/
Sleep 500ms
Screenshot "docs/assets/screenshots/compact.png"
Sleep 500ms
Ctrl+C
Sleep 300ms
