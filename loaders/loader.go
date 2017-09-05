package loaders

type (
  Loader interface {
  }
)

func StaticLoader() Loader {
  return nil
}
