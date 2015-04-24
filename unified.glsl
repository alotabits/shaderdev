#line 1
uniform vec4 viewport;
uniform vec4 cursor;
uniform vec4 time;
uniform mat4 projection;
uniform mat4 view;
uniform mat4 model;

#ifdef VERTEX
	in vec4 position;
	in vec4 color;

	out VertData {
		vec4 position;
		vec4 color;
	} outData;

	void main() {
		gl_Position = projection*view*model*position;
		outData.position = position;
		outData.color = color;
	}
#endif

#ifdef FRAGMENT
	in VertData {
		vec4 position;
		vec4 color;
	} inData;

	out vec4 color;

	void main() {
		color = inData.color;
	}
#endif

