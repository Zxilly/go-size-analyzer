import { useCallback, useState } from "react";
import { useLifecycles } from "react-use";
import { off, on } from "react-use/lib/misc/util";

/**
 * read and write url hash, response to url hash change
 */
export function useHash() {
  const [hash, setHash] = useState(() => window.location.hash);

  const onHashChange = useCallback(() => {
    setHash(window.location.hash);
  }, []);

  useLifecycles(
    () => {
      on(window, "hashchange", onHashChange);
    },
    () => {
      off(window, "hashchange", onHashChange);
    },
  );

  const _setHash = useCallback(
    (newHash: string) => {
      if (newHash !== hash) {
        window.location.hash = newHash;
        if (newHash === "") {
          history.pushState(null, "", " ");
        }
      }
    },
    [hash],
  );

  return [hash, _setHash] as const;
}
