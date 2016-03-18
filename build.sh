#!/bin/bash

rm -rf build/
os=( windows linux )
arch=( 386 amd64 )

for OS in "${os[@]}";
do 
  for ARCH in "${arch[@]}";
  do
    mkdir -p build/"$OS"_"$ARCH"
    cd build/"$OS"_"$ARCH"
    echo `pwd`
    GOOS=$OS GOARCH=$ARCH go build ../../
    cp -r ../../ui ./ui
    cd ../
    zip -r ./"$OS"_"$ARCH".zip ./"$OS"_"$ARCH"/*
    cd ..
  done
done
