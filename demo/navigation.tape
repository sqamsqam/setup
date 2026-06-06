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

Hide
Type "unset NO_COLOR; export TERM=xterm-256color COLORTERM=truecolor CLICOLOR_FORCE=1; clear"
Enter
Show
Type "./bin/setup-linux-amd64 --demo"
Enter
Wait+Screen /Setup Admin Console/
Sleep 800ms
Down 2
Sleep 450ms
Enter
Wait+Screen /Groups/
Sleep 450ms
Space
Sleep 450ms
Enter
Wait+Screen /Target Group/
Sleep 550ms
Type "app"
Sleep 700ms
Enter
Wait+Screen /Review Plan/
Sleep 1500ms
