#ifndef LOUNGE_GTKTHEME_H
#define LOUNGE_GTKTHEME_H

// lounge_theme_colors lists the named colors the active GTK theme defines,
// one "name=r,g,b,a" line each, plus "@name=" and "@dark=" lines. Call it on
// the GTK main thread. The caller frees the result.
char *lounge_theme_colors(void *gtk_widget);

// lounge_theme_watch calls goGtkThemeChanged when the GTK theme or its dark
// preference changes. Call it once on the GTK main thread.
void lounge_theme_watch(void);

#endif
