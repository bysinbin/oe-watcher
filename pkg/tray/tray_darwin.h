#pragma once

void InitTray(const char *initialTitle, int port, int autoSkip);
void UpdateTrayTrack(const char *trackText, int isMatched);
void UpdateTrayAutoSkip(int autoSkip);
void OpenBrowserURL(const char *urlStr);
void RunCocoaApp(void);
