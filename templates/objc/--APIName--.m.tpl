{{template "preHeaderComment" .}}

#import "{{.Config.APIName}}.h"

static const NSString * k{{.Config.APIPrefix}}BaseURL = @"TODO: This should be the default base url";

@interface {{.Config.APIName}} : NSObject
@property (nonatomic, strong) {{.Config.APIPrefix}}ResourceManager *resourceManager
@end

@implementation {{.Config.APIName}}

+ (instancetype)default
{
    {{- $apiVar := lowerFirst .Config.APIName}}
    static {{.Config.APIName}} *{{$apiVar}} = nil;
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        {{$apiVar}} = [[self alloc] init];
    });
    return {{$apiVar}};
}

- (instancetype)init
{
    if (self = [super init])
    {
        _resourceManager = [[{{.Config.APIPrefix}}ResourceManager alloc] initWithBaseURL:k{{.Config.APIPrefix}}BaseURL];
    }
    return self;
}

- (void)useBaseURLString:(NSString *)baseURL
{
    self.resourceManager.baseURL = baseURL
}

- (id<{{.Config.APIPrefix}}Model>)model:(Class<{{.Config.APIPrefix}}Model>)modelClass
{
    NSAssert([modelClass conformsToProtocol:@protocol({{.Config.APIPrefix}}Model)], @"The model class must conform {{.Config.APIPrefix}}Model protocol");
    return [modelClass modelWithResourceManager:self.resourceManager];
}

+ (void)setGlobalErrorHandlerWithBlock:(void (^)(NSError *))block
{
	PMKUnhandledErrorHandler = ^void(NSError *error)
	{
		dispatch_async(dispatch_get_main_queue(), ^
		{
			block(error);
		});
	};
}

@end
