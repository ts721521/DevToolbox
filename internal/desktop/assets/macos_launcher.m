#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include <arpa/inet.h>
#include <netinet/in.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

static const int kHubPort = 17890;

static BOOL hubUp(void) {
	int fd = socket(AF_INET, SOCK_STREAM, 0);
	if (fd < 0) {
		return NO;
	}
	struct sockaddr_in addr;
	memset(&addr, 0, sizeof(addr));
	addr.sin_family = AF_INET;
	addr.sin_port = htons(kHubPort);
	inet_pton(AF_INET, "127.0.0.1", &addr.sin_addr);
	struct timeval tv = {.tv_sec = 0, .tv_usec = 300000};
	setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
	BOOL ok = connect(fd, (struct sockaddr *)&addr, sizeof(addr)) == 0;
	close(fd);
	return ok;
}

static void startHub(NSString *goBin) {
	if (hubUp()) {
		return;
	}
	static NSTimeInterval last = 0;
	NSTimeInterval now = [NSDate timeIntervalSinceReferenceDate];
	if (now - last < 2.0) {
		return;
	}
	last = now;
	if (![[NSFileManager defaultManager] isExecutableFileAtPath:goBin]) {
		return;
	}
	NSTask *task = [[NSTask alloc] init];
	task.executableURL = [NSURL fileURLWithPath:goBin];
	task.arguments = @[];
	task.standardOutput = [NSFileHandle fileHandleWithNullDevice];
	task.standardError = [NSFileHandle fileHandleWithNullDevice];
	[task launchAndReturnError:nil];
}

@interface ToolDockApp : NSObject <NSApplicationDelegate, NSWindowDelegate>
@property(copy) NSString *goBin;
@property(strong) NSWindow *window;
@property(strong) WKWebView *webView;
@end

@implementation ToolDockApp

- (void)ensureWindow {
	if (self.window) {
		[self.window makeKeyAndOrderFront:nil];
		[NSApp activateIgnoringOtherApps:YES];
		return;
	}
	NSRect rect = NSMakeRect(0, 0, 1080, 760);
	NSUInteger style = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable |
			   NSWindowStyleMaskResizable;
	NSWindow *win = [[NSWindow alloc] initWithContentRect:rect
						    styleMask:style
						      backing:NSBackingStoreBuffered
							defer:NO];
	win.title = @"工坞";
	win.delegate = self;
	[win center];
	WKWebView *web = [[WKWebView alloc] initWithFrame:rect];
	win.contentView = web;
	self.webView = web;
	self.window = win;
	[win makeKeyAndOrderFront:nil];
	[NSApp activateIgnoringOtherApps:YES];
}

- (void)loadUI {
	[self ensureWindow];
	NSURL *url = [NSURL URLWithString:@"http://127.0.0.1:17890"];
	[self.webView loadRequest:[NSURLRequest requestWithURL:url]];
}

- (void)showWhenReady {
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		for (int i = 0; i < 25; i++) {
			if (hubUp()) {
				break;
			}
			usleep(200000);
		}
		dispatch_async(dispatch_get_main_queue(), ^{ [self loadUI]; });
	});
}

- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	(void)notification;
	startHub(self.goBin);
	[self showWhenReady];
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)flag {
	(void)sender;
	startHub(self.goBin);
	if (flag && self.window) {
		[self.window makeKeyAndOrderFront:nil];
		[NSApp activateIgnoringOtherApps:YES];
		return NO;
	}
	if (hubUp()) {
		[self loadUI];
	} else {
		[self showWhenReady];
	}
	return NO;
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
	(void)sender;
	return NO;
}

- (BOOL)windowShouldClose:(NSWindow *)sender {
	[sender orderOut:nil];
	return NO;
}

@end

int main(int argc, const char *argv[]) {
	(void)argc;
	(void)argv;
	@autoreleasepool {
		NSString *exe = [[NSBundle mainBundle] executablePath];
		NSString *macos = [exe stringByDeletingLastPathComponent];
		NSString *contents = [macos stringByDeletingLastPathComponent];
		NSString *goBin = [contents stringByAppendingPathComponent:@"Helpers/tooldock"];
		ToolDockApp *app = [[ToolDockApp alloc] init];
		app.goBin = goBin;
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		[NSApp setDelegate:app];
		[NSApp run];
	}
	return 0;
}
