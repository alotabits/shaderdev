#!/bin/sh

go build

./shaderdev vs:vert.glsl vs:unified.glsl fs:frag.glsl fs:unified.glsl
