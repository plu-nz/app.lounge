#include "render.h"

#include <epoxy/egl.h>
#include <epoxy/gl.h>
#include <epoxy/glx.h>
#include <gtk/gtk.h>
#ifdef GDK_WINDOWING_WAYLAND
#include <gdk/wayland/gdkwayland.h>
#endif
#ifdef GDK_WINDOWING_X11
#include <gdk/x11/gdkx.h>
#endif
#include <mpv/render_gl.h>
#include <stdio.h>

static mpv_handle *mpv;
static mpv_render_context *render_ctx;
static GtkWidget *gl_area;

static void *get_proc_address(void *ctx, const char *name) {
	(void)ctx;
	if (epoxy_has_egl()) {
		return (void *)eglGetProcAddress(name);
	}
	return (void *)glXGetProcAddressARB((const GLubyte *)name);
}

static gboolean queue_render(gpointer data) {
	(void)data;
	// Only redraw when mpv has a new frame for us.
	if (gl_area && render_ctx &&
	    (mpv_render_context_update(render_ctx) & MPV_RENDER_UPDATE_FRAME)) {
		gtk_gl_area_queue_render(GTK_GL_AREA(gl_area));
	}
	return G_SOURCE_REMOVE;
}

// mpv calls this from its own thread; hop to the GTK main loop.
static void on_mpv_update(void *ctx) {
	(void)ctx;
	g_idle_add(queue_render, NULL);
}

static void on_realize(GtkGLArea *area, gpointer data) {
	(void)data;
	gtk_gl_area_make_current(area);
	if (gtk_gl_area_get_error(area) != NULL || render_ctx != NULL) {
		return;
	}
	mpv_opengl_init_params gl_init = {.get_proc_address = get_proc_address};
	// Hardware decoding interop (VAAPI over EGL) needs the native display.
	mpv_render_param display = {MPV_RENDER_PARAM_INVALID, NULL};
	GdkDisplay *gdk_display = gtk_widget_get_display(GTK_WIDGET(area));
#ifdef GDK_WINDOWING_WAYLAND
	if (GDK_IS_WAYLAND_DISPLAY(gdk_display)) {
		display = (mpv_render_param){MPV_RENDER_PARAM_WL_DISPLAY,
			gdk_wayland_display_get_wl_display(gdk_display)};
	}
#endif
#ifdef GDK_WINDOWING_X11
	// GTK deprecates raw Xlib access, but mpv's X11 interop needs it.
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
	if (GDK_IS_X11_DISPLAY(gdk_display)) {
		display = (mpv_render_param){MPV_RENDER_PARAM_X11_DISPLAY,
			gdk_x11_display_get_xdisplay(gdk_display)};
	}
#pragma GCC diagnostic pop
#endif
	mpv_render_param params[] = {
		{MPV_RENDER_PARAM_API_TYPE, MPV_RENDER_API_TYPE_OPENGL},
		{MPV_RENDER_PARAM_OPENGL_INIT_PARAMS, &gl_init},
		display,
		{MPV_RENDER_PARAM_INVALID, NULL},
	};
	int err = mpv_render_context_create(&render_ctx, mpv, params);
	if (err < 0) {
		fprintf(stderr, "lounge: mpv render context: %s\n", mpv_error_string(err));
		render_ctx = NULL;
		return;
	}
	mpv_render_context_set_update_callback(render_ctx, on_mpv_update, NULL);
}

static void on_unrealize(GtkGLArea *area, gpointer data) {
	(void)data;
	gtk_gl_area_make_current(area);
	if (render_ctx) {
		mpv_render_context_free(render_ctx);
		render_ctx = NULL;
	}
}

static gboolean on_render(GtkGLArea *area, GdkGLContext *context, gpointer data) {
	(void)area;
	(void)context;
	(void)data;
	if (!render_ctx) {
		glClearColor(0, 0, 0, 1);
		glClear(GL_COLOR_BUFFER_BIT);
		return TRUE;
	}
	// GtkGLArea sets the viewport to its buffer size, which already includes
	// the (possibly fractional) display scale.
	GLint fbo = 0;
	GLint viewport[4] = {0};
	glGetIntegerv(GL_FRAMEBUFFER_BINDING, &fbo);
	glGetIntegerv(GL_VIEWPORT, viewport);
	mpv_opengl_fbo target = {.fbo = fbo, .w = viewport[2], .h = viewport[3]};
	int flip = 1;
	// GTK picks when frames reach the screen, so mpv must not wait for a
	// target time on the main loop ("video-timing-offset" is 0 as well).
	// report_swap is not used: GTK does not tell us when the buffer flips.
	int block = 0;
	mpv_render_param params[] = {
		{MPV_RENDER_PARAM_OPENGL_FBO, &target},
		{MPV_RENDER_PARAM_FLIP_Y, &flip},
		{MPV_RENDER_PARAM_BLOCK_FOR_TARGET_TIME, &block},
		{MPV_RENDER_PARAM_INVALID, NULL},
	};
	mpv_render_context_render(render_ctx, params);
	return TRUE;
}

int lounge_attach(void *gtk_window, mpv_handle *handle) {
	if (gl_area != NULL) {
		return 0;
	}
	mpv = handle;
	GtkWidget *box = gtk_window_get_child(GTK_WINDOW(gtk_window));
	if (box == NULL) {
		return -1;
	}
	GtkWidget *webview = gtk_widget_get_last_child(box);
	if (webview == NULL || !GTK_IS_BOX(box)) {
		return -2;
	}

	g_object_ref(webview);
	gtk_box_remove(GTK_BOX(box), webview);

	gl_area = gtk_gl_area_new();
	gtk_gl_area_set_auto_render(GTK_GL_AREA(gl_area), FALSE);
	gtk_gl_area_set_has_depth_buffer(GTK_GL_AREA(gl_area), FALSE);
	g_signal_connect(gl_area, "realize", G_CALLBACK(on_realize), NULL);
	g_signal_connect(gl_area, "unrealize", G_CALLBACK(on_unrealize), NULL);
	g_signal_connect(gl_area, "render", G_CALLBACK(on_render), NULL);
	gtk_widget_set_can_focus(gl_area, FALSE);

	GtkWidget *overlay = gtk_overlay_new();
	gtk_widget_set_hexpand(overlay, TRUE);
	gtk_widget_set_vexpand(overlay, TRUE);
	gtk_overlay_set_child(GTK_OVERLAY(overlay), gl_area);
	gtk_overlay_add_overlay(GTK_OVERLAY(overlay), webview);
	gtk_box_append(GTK_BOX(box), overlay);
	g_object_unref(webview);

	gtk_widget_grab_focus(webview);
	return 0;
}

void lounge_detach(void) {
	if (gl_area && gtk_widget_get_realized(gl_area)) {
		on_unrealize(GTK_GL_AREA(gl_area), NULL);
	}
}

static void json_string(GString *out, const char *s) {
	g_string_append_c(out, '"');
	for (const unsigned char *p = (const unsigned char *)s; *p; p++) {
		switch (*p) {
		case '"': g_string_append(out, "\\\""); break;
		case '\\': g_string_append(out, "\\\\"); break;
		case '\n': g_string_append(out, "\\n"); break;
		case '\r': g_string_append(out, "\\r"); break;
		case '\t': g_string_append(out, "\\t"); break;
		default:
			if (*p < 0x20) {
				g_string_append_printf(out, "\\u%04x", *p);
			} else {
				g_string_append_c(out, (char)*p);
			}
		}
	}
	g_string_append_c(out, '"');
}

static void node_json(GString *out, const mpv_node *node) {
	switch (node->format) {
	case MPV_FORMAT_STRING:
		json_string(out, node->u.string);
		break;
	case MPV_FORMAT_FLAG:
		g_string_append(out, node->u.flag ? "true" : "false");
		break;
	case MPV_FORMAT_INT64:
		g_string_append_printf(out, "%" G_GINT64_FORMAT, (gint64)node->u.int64);
		break;
	case MPV_FORMAT_DOUBLE: {
		char buf[G_ASCII_DTOSTR_BUF_SIZE];
		g_string_append(out, g_ascii_dtostr(buf, sizeof buf, node->u.double_));
		break;
	}
	case MPV_FORMAT_NODE_ARRAY:
		g_string_append_c(out, '[');
		for (int i = 0; i < node->u.list->num; i++) {
			if (i) g_string_append_c(out, ',');
			node_json(out, &node->u.list->values[i]);
		}
		g_string_append_c(out, ']');
		break;
	case MPV_FORMAT_NODE_MAP:
		g_string_append_c(out, '{');
		for (int i = 0; i < node->u.list->num; i++) {
			if (i) g_string_append_c(out, ',');
			json_string(out, node->u.list->keys[i]);
			g_string_append_c(out, ':');
			node_json(out, &node->u.list->values[i]);
		}
		g_string_append_c(out, '}');
		break;
	default:
		g_string_append(out, "null");
	}
}

char *lounge_node_json(const mpv_node *node) {
	GString *out = g_string_new(NULL);
	node_json(out, node);
	return g_string_free(out, FALSE);
}
