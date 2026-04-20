package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"

	"github.com/HappyLadySauce/TraveLight/cmd/app/options"
	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/auth"
	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/comment"
	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/craw"
	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/ranking"
	"github.com/HappyLadySauce/TraveLight/cmd/app/routes/search"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
	"github.com/HappyLadySauce/TraveLight/pkg/model"
)

func NewAPICommand(ctx context.Context, basename string) *cobra.Command {
	opts := options.NewOptions()
	cmd := &cobra.Command{
		Use:   basename,
		Short: "TraveLight is a web server for TraveLight",
		Long:  "TraveLight is a web server for TraveLight",
		RunE: func(cmd *cobra.Command, args []string) error {
			// bind command line flags to viper (command line args override config file)
			// 从命令行标志中绑定到 viperiper
			if err := viper.BindPFlags(cmd.Flags()); err != nil {
				return fmt.Errorf("failed to bind command line flags: %w", err)
			}

			// unmarshal viper config to options struct
			// 从 viperiper 解析配置到选项结构体
			if err := viper.Unmarshal(opts); err != nil {
				return fmt.Errorf("failed to unmarshal viper config: %w", err)
			}

			// validate options after flags & config are fully populated
			// 验证选项
			if err := opts.Validate(); err != nil {
				klog.ErrorS(err, "Configuration validation failed")
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			// create service context
			// 创建服务上下文
			serviceCtx, err := svc.NewServiceContext(*opts)
			if err != nil {
				klog.ErrorS(err, "Failed to create service context")
				return err
			}

			// ensure service context is closed on exit
			// 确保服务上下文在退出时关闭
			defer func() {
				if err := serviceCtx.Close(); err != nil {
					klog.ErrorS(err, "Failed to close service context")
				}
			}()

			return run(ctx, serviceCtx)
		},
	}

	// Add command line flags
	// 添加命令行标志
	nfs := opts.AddFlags(cmd.Flags(), basename)
	flag.SetUsageAndHelpFunc(cmd, *nfs, 80)

	return cmd
}

func run(ctx context.Context, serviceCtx *svc.ServiceContext) error {
	return serve(ctx, serviceCtx)
}

func serve(ctx context.Context, svcCtx *svc.ServiceContext) error {
	if err := model.AutoMigrate(svcCtx.DB); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	auth.RegisterRoutes(svcCtx)
	comment.RegisterRoutes(svcCtx)
	craw.RegisterRoutes(svcCtx)
	ranking.RegisterRoutes(svcCtx)
	search.RegisterRoutes(svcCtx)

	address := fmt.Sprintf("%s:%d", svcCtx.Config.ServerOptions.BindAddress, svcCtx.Config.ServerOptions.BindPort)
	klog.InfoS("Listening and serving on", "address", address)

	srv := router.NewServer(address)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.Fatal(err)
		}
	}()

	<-ctx.Done()
	klog.InfoS("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		klog.ErrorS(err, "Failed to shutdown server timeout, Server forced to shutdown")
		return err
	}

	klog.InfoS("Server exited gracefully")
	return nil
}
