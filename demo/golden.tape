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

Hide
Type "unset NO_COLOR; export TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1; clear"
Enter
Show
Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
Sleep 900ms
Down 42
Sleep 300ms
Space
Sleep 300ms
Down 7
Space
Sleep 250ms
Down
Space
Sleep 700ms
Enter
Wait+Screen /TIMEZONE/
Sleep 600ms
Enter
Wait+Screen /Target User/
Type "dev"
Sleep 600ms
Enter
Wait+Screen /SSH PUBLIC KEY/
Type "ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAA== setup-demo"
Sleep 800ms
Enter
Wait+Screen /Review Plan/
Sleep 1200ms
Enter
Wait+Screen /Fresh setup complete/
Sleep 2600ms
