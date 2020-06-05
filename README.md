# GoNes ![CI](https://github.com/tiagolobocastro/gones/workflows/Go/badge.svg)
NES Emulator written in go

Uses the 2 libraries from github.com/faiface
2D game library: github.com/faiface/pixel
audio library: github.com/faiface/beep

Also, optionally uses the portaudio library: github.com/gordonklaus/portaudio



---
## Building on Windows
Requirements: Golang, MinGW toolchain, Portaudio

### Using MSYS2 to install mingw and portaudio:
#### Setup mingw64 toolchain
>pacman -S --needed base-devel mingw-w64-i686-toolchain mingw-w64-x86_64-toolchain
#### Add mingw64 binaries to the path
>echo 'export PATH=/mingw64/bin:$PATH' >> ~/.bashrc

#### Install PortAudio
>pacman -S mingw64/mingw-w64-x86_64-portaudio

#### Install golang from https://golang.org/dl/

### Now we're ready
>go get github.com/tiagolobocastro/gones


---
## Building on Linux
Requirements: Golang, X dev, Portaudio
#### Install X devel
Ubuntu:
>apt install libgl1-mesa-dev xorg-dev


Arch:
>pacman -S xorg-server-devel
#### Install PortAudio
Ubuntu:
>apt install portaudio19-dev


Arch:
>pacman -S base-devel portaudio

### Now we're ready
>go get github.com/tiagolobocastro/gones



# Running
>gones --help 

Usage of gones: 

-audio string 
>beep, portaudio or nil (default "beep")

-logaudio 
>log audio sampling average every second (debug) 

-rom string 
>path to the iNes Rom file to run 

-verbose flag
>verbose logs (debug only)

-freerun flag
>run as fast as possible with double buffered sync (debug only)

-spritelimit flag
>limit number of sprites per scanline to 8 (true to the NES)