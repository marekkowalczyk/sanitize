#!/usr/bin/env bash

oname=""
filen=""
exten=""
nname=""

for oname in "$@"
do
	filen=$(echo "${oname%.*}")
	exten=$(echo "${oname##*.}")

	if [ "$exten" = "$filen" ]
		then
			nname=$(sanitize "$filen")
			mv -nv "$oname" "$nname"
		else
			nfilen=$(sanitize "$filen")
			nexten=$(sanitize "$exten")
			nname="$nfilen.$nexten"
			mv -nv "$oname" "$nname"
	fi
done
