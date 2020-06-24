!/usr/bin/env bash

for oname in "$@"; do
	filen=$(echo "${oname%.*}")
	exten=$(echo "${oname##*.}")
	nfilen=$(sanitize "$filen")
	nexten=$(sanitize "$exten")
	nname="$nfilen.$nexten"
	mv -nv "$oname" "$nname"
done
