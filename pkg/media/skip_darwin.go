package media

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#include <dlfcn.h>

void SendNativeMediaNext() {
	// 1. Post system-wide hardware media key event (Next Track)
	NSEvent *event1 = [NSEvent otherEventWithType:NSEventTypeSystemDefined
	                                     location:NSZeroPoint
	                                modifierFlags:(19 << 16) | 0xa00
	                                    timestamp:0
	                                 windowNumber:0
	                                      context:nil
	                                      subtype:8
	                                        data1:(19 << 16) | (0xa << 8)
	                                        data2:-1];
	CGEventPost(kCGHIDEventTap, [event1 CGEvent]);

	NSEvent *event2 = [NSEvent otherEventWithType:NSEventTypeSystemDefined
	                                     location:NSZeroPoint
	                                modifierFlags:(19 << 16) | 0xb00
	                                    timestamp:0
	                                 windowNumber:0
	                                      context:nil
	                                      subtype:8
	                                        data1:(19 << 16) | (0xb << 8)
	                                        data2:-1];
	CGEventPost(kCGHIDEventTap, [event2 CGEvent]);

	// 2. Send MediaRemote kMRNextTrack command
	void *lib = dlopen("/System/Library/PrivateFrameworks/MediaRemote.framework/MediaRemote", RTLD_NOW);
	if (lib) {
		void (*sendCommand)(int, id) = (void (*)(int, id))dlsym(lib, "MRMediaRemoteSendCommand");
		if (sendCommand) {
			sendCommand(4, nil);
		}
	}
}
*/
import "C"

// SendHardwareNextTrack triggers native macOS hardware media key (Next Track) and MediaRemote command.
func SendHardwareNextTrack() {
	C.SendNativeMediaNext()
}
