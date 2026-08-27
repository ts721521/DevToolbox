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

static BOOL focusUI(void) {
	NSString *src = @"set names to {\"Google Chrome\", \"Microsoft Edge\"}\n"
			 @"repeat with appName in names\n"
			 @"try\n"
			 @"tell application \"System Events\"\n"
			 @"if not (exists process (appName as text)) then error \"skip\"\n"
			 @"end tell\n"
			 @"tell application appName\n"
			 @"repeat with w in windows\n"
			 @"try\n"
			 @"set u to URL of active tab of w\n"
			 @"if u contains \"127.0.0.1:17890\" then\n"
			 @"set index of w to 1\n"
			 @"activate\n"
			 @"return \"ok\"\n"
			 @"end if\n"
			 @"end try\n"
			 @"end repeat\n"
			 @"end tell\n"
			 @"end try\n"
			 @"end repeat\n"
			 @"return \"no\"";
	NSAppleScript *as = [[NSAppleScript alloc] initWithSource:src];
	NSAppleEventDescriptor *r = [as executeAndReturnError:nil];
	return r && [[r stringValue] isEqualToString:@"ok"];
}

static void openUI(void) {
	static NSTimeInterval last = 0;
	NSTimeInterval now = [NSDate timeIntervalSinceReferenceDate];
	if (now - last < 1.5) {
		return;
	}
	if (focusUI()) {
		last = now;
		return;
	}
	last = now;
	NSArray<NSString *> *bins = @[
		@"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		@"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	];
	for (NSString *bin in bins) {
		if (![[NSFileManager defaultManager] isExecutableFileAtPath:bin]) {
			continue;
		}
		NSTask *task = [[NSTask alloc] init];
		task.executableURL = [NSURL fileURLWithPath:bin];
		task.arguments = @[ @"--app=http://127.0.0.1:17890", @"--window-size=1080,760" ];
		if ([task launchAndReturnError:nil]) {
			return;
		}
	}
	[[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:@"http://127.0.0.1:17890"]];
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
