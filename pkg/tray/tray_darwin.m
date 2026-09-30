#import <Cocoa/Cocoa.h>
#import "tray_darwin.h"

// Forward declare exported Go functions
extern void goOpenWebPanel(void);
extern void goSkipTrack(void);
extern void goToggleAutoSkip(void);
extern void goSyncAPI(void);
extern void goQuitApp(void);
extern void goTriggerFastPoll(void);

@interface TrayController : NSObject
@property (strong, nonatomic) NSStatusItem *statusItem;
@property (strong, nonatomic) NSMenuItem *statusMenu;
@property (strong, nonatomic) NSMenuItem *trackMenu;
@property (strong, nonatomic) NSMenuItem *autoSkipMenu;

+ (instancetype)shared;
- (void)menuOpenWeb:(id)sender;
- (void)menuSkipTrack:(id)sender;
- (void)menuToggleAutoSkip:(id)sender;
- (void)menuSync:(id)sender;
- (void)menuQuit:(id)sender;
- (void)setupDarwinNotifications;
- (void)onPlayerNotification:(NSNotification *)note;
@end

static TrayController *s_controller = nil;

@implementation TrayController

+ (instancetype)shared {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        s_controller = [[TrayController alloc] init];
    });
    return s_controller;
}

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

- (void)setupDarwinNotifications {
    NSDistributedNotificationCenter *center = [NSDistributedNotificationCenter defaultCenter];
    [center addObserver:self
               selector:@selector(onPlayerNotification:)
                   name:@"com.spotify.client.PlaybackStateChanged"
                 object:nil];
    [center addObserver:self
               selector:@selector(onPlayerNotification:)
                   name:@"com.apple.Music.playerInfo"
                 object:nil];
}

- (void)onPlayerNotification:(NSNotification *)note {
    goTriggerFastPoll();
}

@end

void InitTray(const char *initialTitle, int port, int autoSkip) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    TrayController *tc = [TrayController shared];
    [tc setupDarwinNotifications];

    // In macOS 10.14+, callers must hold a strong reference to NSStatusItem
    tc.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];

    NSString *titleStr = @"🎧 OE";
    if (initialTitle != NULL) {
        NSString *s = [NSString stringWithUTF8String:initialTitle];
        if (s) titleStr = s;
    }
    tc.statusItem.button.title = titleStr;

    NSMenu *menu = [[NSMenu alloc] initWithTitle:@"OE Watcher"];

    // Header
    NSMenuItem *header = [[NSMenuItem alloc] initWithTitle:@"OE LIST // MEDYA TAKİPÇİSİ" action:nil keyEquivalent:@""];
    [header setEnabled:NO];
    [menu addItem:header];

    // Status
    tc.statusMenu = [[NSMenuItem alloc] initWithTitle:@"Durum: Canlı İzleniyor" action:nil keyEquivalent:@""];
    [tc.statusMenu setEnabled:NO];
    [menu addItem:tc.statusMenu];

    // Current Track
    tc.trackMenu = [[NSMenuItem alloc] initWithTitle:@"Çalan: -" action:nil keyEquivalent:@""];
    [tc.trackMenu setEnabled:NO];
    [menu addItem:tc.trackMenu];

    [menu addItem:[NSMenuItem separatorItem]];

    // Web Panel Button
    NSString *webTitle = [NSString stringWithFormat:@"🌐 Web Panelini Aç (Port: %d)", port];
    NSMenuItem *webItem = [[NSMenuItem alloc] initWithTitle:webTitle action:@selector(menuOpenWeb:) keyEquivalent:@"w"];
    [webItem setTarget:tc];
    [menu addItem:webItem];

    // Skip Track Button
    NSMenuItem *skipItem = [[NSMenuItem alloc] initWithTitle:@"⏩ Şarkıyı Hemen Geç" action:@selector(menuSkipTrack:) keyEquivalent:@"s"];
    [skipItem setTarget:tc];
    [menu addItem:skipItem];

    // Auto-Skip Toggle Button
    NSString *autoSkipTitle = autoSkip ? @"⚡ Otomatik Geçiş: [AÇIK]" : @"⚡ Otomatik Geçiş: [KAPALI]";
    tc.autoSkipMenu = [[NSMenuItem alloc] initWithTitle:autoSkipTitle action:@selector(menuToggleAutoSkip:) keyEquivalent:@"a"];
    [tc.autoSkipMenu setTarget:tc];
    [menu addItem:tc.autoSkipMenu];

    // Sync API Button
    NSMenuItem *syncItem = [[NSMenuItem alloc] initWithTitle:@"🔄 API Senkronize Et" action:@selector(menuSync:) keyEquivalent:@"r"];
    [syncItem setTarget:tc];
    [menu addItem:syncItem];

    [menu addItem:[NSMenuItem separatorItem]];

    // Quit Button
    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"❌ Çıkış Yap" action:@selector(menuQuit:) keyEquivalent:@"q"];
    [quitItem setTarget:tc];
    [menu addItem:quitItem];

    tc.statusItem.menu = menu;
}

void UpdateTrayTrack(const char *trackText, int isMatched) {
    if (!trackText) return;

    NSString *str = [NSString stringWithUTF8String:trackText];
    if (!str) {
        str = [NSString stringWithCString:trackText encoding:NSISOLatin1StringEncoding];
    }
    if (!str || [str length] == 0) {
        str = @"-";
    }

    NSString *safeTrackStr = [str copy];

    dispatch_async(dispatch_get_main_queue(), ^{
        TrayController *tc = [TrayController shared];
        if (tc.trackMenu) {
            tc.trackMenu.title = [NSString stringWithFormat:@"Çalan: %@", safeTrackStr];
        }
        if (tc.statusItem && tc.statusItem.button) {
            if (isMatched) {
                tc.statusItem.button.title = @"🚨 OE [UYARI]";
            } else {
                tc.statusItem.button.title = @"🎧 OE";
            }
        }
    });
}

void UpdateTrayAutoSkip(int autoSkip) {
    dispatch_async(dispatch_get_main_queue(), ^{
        TrayController *tc = [TrayController shared];
        if (tc.autoSkipMenu) {
            tc.autoSkipMenu.title = autoSkip ? @"⚡ Otomatik Geçiş: [AÇIK]" : @"⚡ Otomatik Geçiş: [KAPALI]";
        }
    });
}

void OpenBrowserURL(const char *urlStr) {
    if (!urlStr) return;
    NSString *str = [NSString stringWithUTF8String:urlStr];
    if (!str) return;

    NSString *safeURL = [str copy];
    dispatch_async(dispatch_get_main_queue(), ^{
        [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:safeURL]];
    });
}

void RunCocoaApp(void) {
    [NSApp run];
}
