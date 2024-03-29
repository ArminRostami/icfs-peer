package ipfs

import (
	"context"
	"icfs-peer/domain"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	"github.com/pkg/errors"
)

const swarmKey = "swarm.key"

type IpfsService struct {
	repoPath string
	ctx      context.Context
	node     *core.IpfsNode
	userConf *domain.UserConfig
}

func NewService(conf *domain.UserConfig) (context.CancelFunc, *IpfsService, error) {
	pr, err := config.PathRoot()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get default config path")
	}
	if err := setupPlugins(pr); err != nil {
		return nil, nil, errors.Wrap(err, "failed to setup plugins")
	}
	ctx, cancel := context.WithCancel(context.Background())

	return cancel, &IpfsService{ctx: ctx, repoPath: pr, userConf: conf}, nil
}

func (s *IpfsService) Start(ec chan<- error) {
	err := s.setupRepo()
	if err != nil {
		ec <- errors.Wrap(err, "failed to start ipfs service")
		return
	}

	err = s.createNode()
	if err != nil {
		ec <- errors.Wrap(err, "failed to spawn default node")
		return
	}

	opts := []corehttp.ServeOption{
		corehttp.GatewayOption(false, "/ipfs", "/ipns"),
		corehttp.CommandsOption(s.cmdCtx()),
	}

	ec <- corehttp.ListenAndServe(s.node, "/ip4/127.0.0.1/tcp/5001", opts...)
}

func (s *IpfsService) createNode() error {
	repo, err := fsrepo.Open(s.repoPath)
	if err != nil {
		return errors.Wrap(err, "failed to open repo")
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}

	node, err := core.NewNode(s.ctx, nodeOptions)
	if err != nil {
		return errors.Wrap(err, "failed to start new node")
	}
	s.node = node
	return nil
}

func (s *IpfsService) cmdCtx() commands.Context {
	return commands.Context{
		ConfigRoot: s.repoPath,
		LoadConfig: func(path string) (*config.Config, error) {
			return s.node.Repo.Config()
		},
		ConstructNode: func() (*core.IpfsNode, error) {
			return s.node, nil
		},
		ReqLog: &commands.ReqLog{},
	}
}

func (s *IpfsService) setupRepo() error {
	if fsrepo.IsInitialized(s.repoPath) {
		return s.validateRepo()
	}
	return s.initRepo()
}

func (s *IpfsService) validateRepo() error {
	cfg, err := fsrepo.ConfigAt(s.repoPath)
	if err != nil {
		return errors.Wrap(err, "failed to open config file")
	}

	bootMatches := strings.EqualFold(cfg.Bootstrap[0], s.userConf.Bootstrap)
	if !bootMatches {
		log.Println("bootstrap mismatch. fixing...")
		setBootstrap(cfg, s.userConf.Bootstrap)
	}

	key, err := os.ReadFile(path.Join(s.repoPath, swarmKey))
	if err != nil {
		return errors.Wrap(err, "failed to read swarm key")
	}

	keyMatches := strings.EqualFold(string(key), s.userConf.SwarmKey)
	if !keyMatches {
		log.Println("swarm key mismatch. fixing...")

		err := os.Remove(path.Join(s.repoPath, swarmKey))
		if err != nil {
			return errors.Wrap(err, "failed to remove old swarm key")
		}
		err = writeSwarmKey(s.userConf.SwarmKey, s.repoPath)
		if err != nil {
			return errors.Wrap(err, "failed to write new swarm key")
		}
	}

	return nil
}

func (s *IpfsService) initRepo() error {
	log.Printf("setting up new repo at %s\n", s.repoPath)
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return errors.Wrap(err, "failed to init config")
	}
	setupCORS(cfg)

	if err = setBootstrap(cfg, s.userConf.Bootstrap); err != nil {
		return errors.Wrap(err, "failed to set bootstrap")
	}

	if err = fsrepo.Init(s.repoPath, cfg); err != nil {
		return errors.Wrap(err, "failed to init repo")
	}

	err = writeSwarmKey(s.userConf.SwarmKey, s.repoPath)
	if err != nil {
		return errors.Wrap(err, "failed to copy swarm.key file")
	}

	return nil
}

func setupCORS(cfg *config.Config) {
	cfg.API.HTTPHeaders["Access-Control-Allow-Origin"] = []string{
		"*",
	}
	cfg.API.HTTPHeaders["Access-Control-Allow-Methods"] = []string{"POST", "PUT", "GET", "DELETE"}
}

func setBootstrap(cfg *config.Config, bootStr string) error {
	peers, err := config.ParseBootstrapPeers([]string{bootStr})
	if err != nil {
		return errors.Wrap(err, "failed to parse peerAddr")
	}

	cfg.SetBootstrapPeers(peers)
	return nil
}

func writeSwarmKey(key, repoPath string) error {
	if err := os.WriteFile(path.Join(repoPath, swarmKey), []byte(key), 0644); err != nil {
		return errors.Wrap(err, "failed to write to file")
	}
	return nil
}

func setupPlugins(externalPluginsPath string) error {
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return errors.Wrap(err, "failed to load plugins")
	}

	if err := plugins.Initialize(); err != nil {
		return errors.Wrap(err, "failed to init plugins")
	}

	if err := plugins.Inject(); err != nil {
		return errors.Wrap(err, "failed to inject plugins")
	}

	return nil
}
