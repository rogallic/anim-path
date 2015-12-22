# anim-path
Creating simple incrase animation for svg path on map.

##For convert images to video:
$ sudo apt-get install libav-tools
$ avconv -framerate 25 -f image2 -i results/%d.png -c:v h264 -crf 1 result.mov