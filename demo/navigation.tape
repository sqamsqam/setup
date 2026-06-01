Output "docs/assets/gifs/navigation.gif"

Set Shell "bash"
Set Width 1000
Set Height 680
Set FontSize 16
Set Padding 12
Set Theme "Dracula"
Set WindowBar "Colorful"
Set WindowBarSize 32
Set BorderRadius 8
Set TypingSpeed 18ms
Set Framerate 24

Type "env -u NO_COLOR TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1 ./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
Sleep 500ms
Down 4
Sleep 300ms
Space
Sleep 300ms
Down 7
Sleep 300ms
Space
Sleep 300ms
Enter
Wait+Screen /TIMEZONE/
Sleep 300ms
Backspace 3
Type "new york"
Sleep 300ms
Tab
Sleep 300ms
Enter
Wait+Screen /Target User/
Type "dev"
Sleep 300ms
Enter
Wait+Screen /SSH PUBLIC KEY/
Type "ssh-ed25519 AAAAAAAAAAAAAAAAAAAAAA== setup-demo"
Sleep 300ms
Enter
Wait+Screen /Review Plan/
Sleep 1s
