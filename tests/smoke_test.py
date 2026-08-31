import subprocess
import sys


def test_module_starts() -> None:
    subprocess.run([sys.executable, "-m", "semantic_cache"], check=True)
