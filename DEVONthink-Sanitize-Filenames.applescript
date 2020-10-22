tell application id "DNtp"
    repeat with thisRecord in (selection as list)
    	set theName to the name of thisRecord as text
    	set the comment of thisRecord to theName
       	set theNewName to do shell script "~/go/bin/sanitize " & "'" & theName & "'"
    	set the name of thisRecord to theNewName
    end repeat
end tell
