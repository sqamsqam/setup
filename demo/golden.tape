Output "docs/assets/golden-demo.gif"

Set Shell "bash"
Set Width 1100
Set Height 840
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 18ms
Set Framerate 24

Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
Sleep 600ms
Down 11
Sleep 200ms
Space
Sleep 200ms
Down 8
Space
Down
Space
Sleep 500ms
Enter
Wait+Screen /Timezone/
Sleep 300ms
Enter
Wait+Screen /Target User/
Type "dev"
Sleep 300ms
Enter
Wait+Screen /SSH public key/
Type "ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAA== setup-demo"
Sleep 500ms
Enter
Wait+Screen /Review Plan/
Sleep 800ms
Enter
Wait+Screen /Fresh setup complete/
Sleep 2s
