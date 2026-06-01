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

Type "env -u NO_COLOR TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1 ./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Fresh Ubuntu Instance Setup/
Enter
Wait+Screen /TIMEZONE/
Enter
Wait+Screen /Target User/
Type "Root User"
Sleep 400ms
Enter
Wait+Screen /invalid username/
Sleep 500ms
Screenshot "docs/assets/screenshots/error.png"
Sleep 1s
