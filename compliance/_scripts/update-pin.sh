#!/bin/bash


# Required arguments
# LABEL: the label to look for in the glide.yaml file
if [[ -z "$LABEL" ]]; then echo "missing required variable LABEL"; exit 1; fi
# REPO: the repo to pull from
if [[ -z "$REPO" ]]; then echo "missing required variable REPO"; exit 1; fi 
# VERSION: the version to pull from the repo
if [[ -z "$VERSION" ]]; then echo "missing required variable VERSION"; exit 1; fi
# DEFAULT_REPO: the primary upstream repo for this project
if [[ -z "$DEFAULT_REPO" ]]; then echo "missing required variable DEFAULT_REPO"; exit 1; fi
# BRANCH: the branch to pull from the repo
if [[ -z "$BRANCH" ]]; then	echo "missing required variable BRANCH"; exit 1; fi
# GLIDE: the path to the glide.yaml file
if [[ -z "$GLIDE" ]]; then echo "missing required variable GLIDE"; exit 1; fi


echo "Updating pins for:"
echo "$LABEL"  #LIBRARY_GLIDE_LABEL
echo "$REPO" #LIBRARY_REPO
echo "$VERSION" #LIBRARY_VERSION
echo "$DEFAULT_REPO"
echo "$BRANCH"
echo "$GLIDE"  #glide.yaml

echo "Updating pin for $LABEL to $VERSION from $REPO"; 


#pull out the lines after our label to look at
L=$(grep -A 30 $LABEL $GLIDE);
#locate the first instance of the strings "package" and "version"
P=$(echo $L | grep -b -o 'package:' | head -2 | tail -n1 | cut -f1 -d:); 
V=$(echo $L | grep -b -o 'version:' | head -1 | cut -f1 -d:); 
E=$(echo $L | grep -b -o '.$' | cut -f1 -d:);

#echo "P: $P   V: $V  E: $E"
#if "package" occurs before "version" or the end insert a version line
if [[ $V -gt $P ]] || [[ $V -gt $E ]] ; then 
 	sed -i -r "\|package:[[:print:]]*$LABEL|a\ \ version: MissingLibraryVersion" "$GLIDE"; 
fi; 

OLD_VER=$(grep -A 30 $LABEL $GLIDE |grep --max-count=1 --only-matching --perl-regexp "version:\s*\K[^\s]+") ;

echo "Old version: $OLD_VER";
echo "New version: $VERSION";
echo "Repo: github.com:$DEFAULT_REPO $BRANCH" ; 

if [[ "$VERSION" != "$OLD_VER" ]]; then 
     sed -i "s|$OLD_VER|$VERSION|" "$GLIDE";
     if [ $REPO != "github.com/$DEFAULT_REPO" ]; then 
       glide mirror set https://github.com/$DEFAULT_REPO $REPO --vcs git; echo "GLIDE MIRRORS UPDATED"; glide mirror list; 
     fi;
   glide up --strip-vendor || glide up --strip-vendor; 
 else 
   echo "No change to $LABEL."; 
fi;