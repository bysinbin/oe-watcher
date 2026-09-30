#import <Cocoa/Cocoa.h>
#import "tray_darwin.h"

// Forward declare exported Go functions
extern void goOpenWebPanel(void);
extern void goSkipTrack(void);
extern void goToggleAutoSkip(void);
extern void goSyncAPI(void);
extern void goQuitApp(void);

@interface TrayBridge : NSObject
- (void)menuOpenWeb:(id)sender;
- (void)menuSkipTrack:(id)sender;
- (void)menuToggleAutoSkip:(id)sender;
- (void)menuSync:(id)sender;
- (void)menuQuit:(id)sender;
@end

static TrayBridge *g_bridge = nil;
static NSStatusItem *g_statusItem = nil;
static NSMenuItem *g_statusMenu = nil;
static NSMenuItem *g_trackMenu = nil;
static NSMenuItem *g_autoSkipMenu = nil;

@implementation TrayBridge
- (void)menuOpenWeb:(id)sender {
    goOpenWebPanel();
}
- (void)menuSkipTrack:(id)sender {
    goSkipTrack();
}
- (void)menuToggleAutoSkip:(id)sender {
    goToggleAutoSkip();
}
- (void)menuSync:(id)sender {
    goSyncAPI();
}
- (void)menuQuit:(id)sender {
    goQuitApp();
    [NSApp terminate:nil];
}
@end

void InitTray(const char *initialTitle, int port, int autoSkip) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        // Hide dock icon so app runs silently in menu bar
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        g_bridge = [[TrayBridge alloc] init];

        g_statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
        g_statusItem.button.title = [NSString stringWithUTF8String:initialTitle];

        NSMenu *menu = [[NSMenu alloc] initWithTitle:@"OE Watcher"];

        // Header
        NSMenuItem *header = [[NSMenuItem alloc] initWithTitle:@"OE LIST // MEDYA TAKİPÇİSİ" action:nil keyEquivalent:@""];
        [header setEnabled:NO];
        [menu addItem:header];

        // Status
        g_statusMenu = [[NSMenuItem alloc] initWithTitle:@"Durum: Canlı İzleniyor" action:nil keyEquivalent:@""];
        [g_statusMenu setEnabled:NO];
        [menu addItem:g_statusMenu];

        // Current Track
        g_trackMenu = [[NSMenuItem alloc] initWithTitle:@"Çalan: -" action:nil keyEquivalent:@""];
        [g_trackMenu setEnabled:NO];
        [menu addItem:g_trackMenu];

        [menu addItem:[NSMenuItem separatorItem]];

        // Web Panel Button
        NSString *webTitle = [NSString stringWithFormat:@"🌐 Web Panelini Aç (Port: %d)", port];
        NSMenuItem *webItem = [[NSMenuItem alloc] initWithTitle:webTitle action:@selector(menuOpenWeb:) keyEquivalent:@"w"];
        [webItem setTarget:g_bridge];
        [menu addItem:webItem];

        // Skip Track Button
        NSMenuItem *skipItem = [[NSMenuItem alloc] initWithTitle:@"⏩ Şarkıyı Hemen Geç" action:@selector(menuSkipTrack:) keyEquivalent:@"s"];
        [skipItem setTarget:g_bridge];
        [menu addItem:skipItem];

        // Auto-Skip Toggle Button
        NSString *autoSkipTitle = autoSkip ? @"⚡ Otomatik Geçiş: [AÇIK]" : @"⚡ Otomatik Geçiş: [KAPALI]";
        g_autoSkipMenu = [[NSMenuItem alloc] initWithTitle:autoSkipTitle action:@selector(menuToggleAutoSkip:) keyEquivalent:@"a"];
        [g_autoSkipMenu setTarget:g_bridge];
        [menu addItem:g_autoSkipMenu];

        // Sync API Button
        NSMenuItem *syncItem = [[NSMenuItem alloc] initWithTitle:@"🔄 API Senkronize Et" action:@selector(menuSync:) keyEquivalent:@"r"];
        [syncItem setTarget:g_bridge];
        [menu addItem:syncItem];

        [menu addItem:[NSMenuItem separatorItem]];

        // Quit Button
        NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"❌ Çıkış Yap" action:@selector(menuQuit:) keyEquivalent:@"q"];
        [quitItem setTarget:g_bridge];
        [menu addItem:quitItem];

        g_statusItem.menu = menu;
    }
}

void UpdateTrayTrack(const char *trackText, int isMatched) {
    NSString *nsText = [NSString stringWithUTF8String:trackText];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (g_trackMenu) {
            g_trackMenu.title = [NSString stringWithFormat:@"Çalan: %@", nsText];
        }
        if (g_statusItem && g_statusItem.button) {
            if (isMatched) {
                g_statusItem.button.title = @"🚨 OE [UYARI]";
            } else {
                g_statusItem.button.title = @"🎧 OE";
            }
        }
    });
}

void UpdateTrayAutoSkip(int autoSkip) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (g_autoSkipMenu) {
            g_autoSkipMenu.title = autoSkip ? @"⚡ Otomatik Geçiş: [AÇIK]" : @"⚡ Otomatik Geçiş: [KAPALI]";
        }
    });
}

void OpenBrowserURL(const char *urlStr) {
    NSString *str = [NSString stringWithUTF8String:urlStr];
    dispatch_async(dispatch_get_main_queue(), ^{
        [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:str]];
    });
}

void RunCocoaApp(void) {
    @autoreleasepool {
        [NSApp run];
    }
}
