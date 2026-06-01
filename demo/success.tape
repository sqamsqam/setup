Output "docs/assets/gifs/success.gif"

Set Shell "bash"
Set Width 1180
Set Height 840
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 12ms
Set Framerate 24

Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
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
Wait+Screen /Review Plan/
Enter
Wait+Screen /Fresh setup complete/
Sleep 800ms
Screenshot "docs/assets/screenshots/success.png"
Sleep 1s
