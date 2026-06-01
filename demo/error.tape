Output "docs/assets/gifs/error.gif"

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

Hide
Type "unset NO_COLOR; export TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1; clear"
Enter
Show
Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
Sleep 700ms
Enter
Wait+Screen /TIMEZONE/
Sleep 500ms
Enter
Wait+Screen /Target User/
Sleep 500ms
Type "Root User"
Sleep 650ms
Enter
Wait+Screen /invalid username/
Sleep 900ms
Screenshot "docs/assets/screenshots/error.png"
Sleep 1400ms
