//go:build darwin && cgo

package app

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <objc/runtime.h>

extern void goOnAppTerminate(void);

static NSWindow *gWindow = nil;
static WKWebView *gWebView = nil;
static NSString *gUIURL = nil;

static void setupMainMenu(void) {
    NSMenu *menubar = [[NSMenu alloc] init];
    NSMenuItem *appMenuItem = [[NSMenuItem alloc] init];
    [menubar addItem:appMenuItem];
    [NSApp setMainMenu:menubar];

    NSMenu *appMenu = [[NSMenu alloc] init];
    NSString *appName = @"MacHole";

    NSMenuItem *hideItem = [[NSMenuItem alloc] initWithTitle:[NSString stringWithFormat:@"Hide %@", appName]
                                                      action:@selector(hide:)
                                               keyEquivalent:@"h"];
    [appMenu addItem:hideItem];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:[NSString stringWithFormat:@"Quit %@", appName]
                                                      action:@selector(terminate:)
                                               keyEquivalent:@"q"];
    [appMenu addItem:quitItem];
    [appMenuItem setSubmenu:appMenu];
}

static void handleDidFinishLaunching(id self, SEL _cmd, NSNotification *notification) {
    setupMainMenu();

    NSRect frame = NSMakeRect(0, 0, 820, 660);
    NSUInteger style = NSWindowStyleMaskTitled |
                       NSWindowStyleMaskClosable |
                       NSWindowStyleMaskMiniaturizable |
                       NSWindowStyleMaskResizable;

    gWindow = [[NSWindow alloc] initWithContentRect:frame
                                          styleMask:style
                                            backing:NSBackingStoreBuffered
                                              defer:NO];
    [gWindow setTitle:@"MacHole — Hardware-Bound SOCKS5 Proxy"];
    [gWindow setMinSize:NSMakeSize(680, 540)];
    [gWindow center];
    [gWindow setDelegate:self];

    [gWindow setAppearance:[NSAppearance appearanceNamed:NSAppearanceNameDarkAqua]];
    [gWindow setBackgroundColor:[NSColor colorWithSRGBRed:0.086 green:0.086 blue:0.094 alpha:1.0]];

    WKWebViewConfiguration *config = [[WKWebViewConfiguration alloc] init];
    gWebView = [[WKWebView alloc] initWithFrame:[[gWindow contentView] bounds] configuration:config];
    [gWebView setAutoresizingMask:(NSViewWidthSizable | NSViewHeightSizable)];
    [gWebView setValue:@(NO) forKey:@"drawsBackground"];

    NSURL *url = [NSURL URLWithString:gUIURL];
    NSURLRequest *request = [NSURLRequest requestWithURL:url];
    [gWebView loadRequest:request];

    [[gWindow contentView] addSubview:gWebView];
    [gWindow makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

static BOOL handleShouldTerminateAfterLastWindowClosed(id self, SEL _cmd, NSApplication *sender) {
    return YES;
}

static void handleWillTerminate(id self, SEL _cmd, NSNotification *notification) {
    goOnAppTerminate();
}

static void RunNativeMacApp(const char *urlStr) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        [app setActivationPolicy:NSApplicationActivationPolicyRegular];

        gUIURL = [NSString stringWithUTF8String:urlStr];

        Class delClass = objc_lookUpClass("MacHoleAppDelegate");
        if (!delClass) {
            delClass = objc_allocateClassPair([NSObject class], "MacHoleAppDelegate", 0);
            class_addMethod(delClass, @selector(applicationDidFinishLaunching:), (IMP)handleDidFinishLaunching, "v@:@");
            class_addMethod(delClass, @selector(applicationShouldTerminateAfterLastWindowClosed:), (IMP)handleShouldTerminateAfterLastWindowClosed, "B@:@");
            class_addMethod(delClass, @selector(applicationWillTerminate:), (IMP)handleWillTerminate, "v@:@");
            objc_registerClassPair(delClass);
        }

        id delegate = [[delClass alloc] init];
        [app setDelegate:delegate];
        [app run];
    }
}
*/
import "C"

import "unsafe"

var globalTerminateCallback func()

//export goOnAppTerminate
func goOnAppTerminate() {
	if globalTerminateCallback != nil {
		globalTerminateCallback()
	}
}

// HasNativeGUI reports whether Cocoa WKWebView GUI is compiled in.
func HasNativeGUI() bool {
	return true
}

// RunPlatformGUI starts the native macOS AppKit run loop.
func RunPlatformGUI(uiURL string, onTerminate func()) {
	globalTerminateCallback = onTerminate
	cURL := C.CString(uiURL)
	defer C.free(unsafe.Pointer(cURL))
	C.RunNativeMacApp(cURL)
}
