#import <Cocoa/Cocoa.h>
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

static void openUI(void) {
	NSURL *url = [NSURL URLWithString:@"http://127.0.0.1:17890"];
	NSArray<NSString *> *chromes = @[
		@"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		@"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	];
	for (NSString *bin in chromes) {
		if (![[NSFileManager defaultManager] isExecutableFileAtPath:bin]) {
			continue;
		}
		NSTask *task = [[NSTask alloc] init];
		task.executableURL = [NSURL fileURLWithPath:bin];
		task.arguments = @[ @"--app=http://127.0.0.1:17890", @"--window-size=1080,760" ];
		NSError *err = nil;
		if ([task launchAndReturnError:&err]) {
			return;
		}
	}
	[[NSWorkspace sharedWorkspace] openURL:url];
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

static void openUIWhenReady(void) {
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
		for (int i = 0; i < 25; i++) {
			if (hubUp()) {
				break;
			}
			usleep(200000);
		}
		dispatch_async(dispatch_get_main_queue(), ^{ openUI(); });
	});
}

@interface ToolDockApp : NSObject <NSApplicationDelegate>
@property(copy) NSString *goBin;
@end

@implementation ToolDockApp
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	(void)notification;
	startHub(self.goBin);
	openUIWhenReady();
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)flag {
	(void)sender;
	(void)flag;
	startHub(self.goBin);
	if (hubUp()) {
		openUI();
	} else {
		openUIWhenReady();
	}
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
