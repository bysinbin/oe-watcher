//go:build darwin

package media

/*
#cgo CFLAGS: -fobjc-arc -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#include <dlfcn.h>

void SendNativeMediaNext() {
	// 1. Post system-wide hardware media key event (Next Track: 19)
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

	// 2. Send MediaRemote kMRNextTrack (4) command
	void *lib = dlopen("/System/Library/PrivateFrameworks/MediaRemote.framework/MediaRemote", RTLD_NOW);
	if (lib) {
		void (*sendCommand)(int, id) = (void (*)(int, id))dlsym(lib, "MRMediaRemoteSendCommand");
		if (sendCommand) {
			sendCommand(4, nil);
		}
	}
}

void SendNativeMediaPlayPause() {
	// 1. Post system-wide hardware media key event (Play/Pause: 16)
	NSEvent *event1 = [NSEvent otherEventWithType:NSEventTypeSystemDefined
	                                     location:NSZeroPoint
	                                modifierFlags:(16 << 16) | 0xa00
	                                    timestamp:0
	                                 windowNumber:0
	                                      context:nil
	                                      subtype:8
	                                        data1:(16 << 16) | (0xa << 8)
	                                        data2:-1];
	CGEventPost(kCGHIDEventTap, [event1 CGEvent]);

	NSEvent *event2 = [NSEvent otherEventWithType:NSEventTypeSystemDefined
	                                     location:NSZeroPoint
	                                modifierFlags:(16 << 16) | 0xb00
	                                    timestamp:0
	                                 windowNumber:0
	                                      context:nil
	                                      subtype:8
	                                        data1:(16 << 16) | (0xb << 8)
	                                        data2:-1];
	CGEventPost(kCGHIDEventTap, [event2 CGEvent]);

	// 2. Send MediaRemote kMRTogglePlayPause (2) command
	void *lib = dlopen("/System/Library/PrivateFrameworks/MediaRemote.framework/MediaRemote", RTLD_NOW);
	if (lib) {
		void (*sendCommand)(int, id) = (void (*)(int, id))dlsym(lib, "MRMediaRemoteSendCommand");
		if (sendCommand) {
			sendCommand(2, nil);
		}
	}
}
*/
import "C"

// SendHardwareNextTrack triggers native macOS hardware media key (Next Track) and MediaRemote command.
func SendHardwareNextTrack() {
	C.SendNativeMediaNext()
}

// SendHardwarePlayPause triggers native macOS hardware media key (Play/Pause) and MediaRemote command.
func SendHardwarePlayPause() {
	C.SendNativeMediaPlayPause()
}
