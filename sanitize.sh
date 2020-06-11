#!/usr/bin/env bash

clean=`printf "$@" | \
	iconv -f UTF8-MAC -t ASCII//TRANSLIT | \
	tr '[:upper:]' '[:lower:]' | \
	sed -E " \
		s=\W+=-=g ;
		s='==g ;
		s=\.=-=g ;
		s=\\\\\=-=g ;
		s=\W+=-=g ;
		s= +=-=g ;
		s/_+/-/g ;
		s/^-+// ;
		s/-+$//
		" |
		tr -s '[:punct:]'
`
echo $clean
