# VHS tape — SFTP browser demo
# Generate: vhs docs/vhs/sftp.tape
# Output:   docs/vhs/sftp.gif
#
# Prerequisites:
#   - tamagosh installed in PATH
#   - at least one connection configured + reachable

Output docs/vhs/sftp.gif

Set FontSize 14
Set Width 1600
Set Height 900
Set Theme "GruvboxDark"
Set TypingSpeed 70ms
Set Padding 20

Hide
Type "tamagosh"
Enter
Sleep 1s
Show

# open SFTP on cursor connection
Sleep 1s
Type "f"

# wait for SFTP to connect + render
Sleep 3s

# navigate local pane
Down
Sleep 400ms
Down
Sleep 400ms
Right
Sleep 1s
Left
Sleep 1s

# switch to remote pane
Tab
Sleep 1s
Down
Sleep 400ms
Down
Sleep 400ms

# show help overlay
Type "h"
Sleep 3s
Escape
Sleep 800ms

# show file info
Type "i"
Sleep 2s
Escape
Sleep 800ms

# back to list
Type "q"
Sleep 1s
Type "q"
Sleep 400ms
