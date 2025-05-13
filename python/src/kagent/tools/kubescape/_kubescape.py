from typing import Annotated, List, Optional

from autogen_core.tools import FunctionTool

from .._utils import create_typed_fn_tool
from ..common import run_command


async def _scan_workload(
    ns: Annotated[Optional[str], "The namespace of the pod to get proxy configuration for"] = None,
    workload_name: Annotated[
        str,
        "name of the pod",
    ] = None,
) -> str:
    return _run_kubescape_command(f"scan workload  {workload_name} {'-n ' + ns if ns else ''}")


kubescape_scan_workload = FunctionTool(
    _scan_workload,
    description="""
This command scans given workload in the namespace for security issues.
""",
    name="kubescape_scan_workload",
)

ScanWorkload = create_typed_fn_tool(kubescape_scan_workload, "kagent.tools.kubescape.ScanWorkload", "ScanWorkload")


# Function that runs the istioctl command in the shell
def _run_kubescape_command(command: str) -> str:
    return run_command("kubescape", command.split(" "))
