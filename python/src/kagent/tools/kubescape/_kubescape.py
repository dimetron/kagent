from typing import Annotated, List, Optional

from autogen_core.tools import FunctionTool

from .._utils import create_typed_fn_tool
from ..common import run_command


async def _kubescape_scan(
    ns: Annotated[Optional[str], "The namespace of the pod to get proxy configuration for"] = None,)-> str:
        return _run_kubescape_command(f"scan framework nsa --verbose {'--namespace ' + ns if ns else ''}")


async def _kubescape_scan_workload(
    ns: Annotated[Optional[str], "The namespace of the workload to scan"] = None,
    workload_name: Annotated[str,"name of workload",    ] = None,) -> str:
    return _run_kubescape_command(f"scan workload  {workload_name} {'-n ' + ns if ns else ''}")


kubescape_scan = FunctionTool(
    _kubescape_scan,
    description="""Scan kubernetes for misconfigurations and vulnerabilities using Kubescape.""",
    name="_kubescape_scan",
)


kubescape_scan_workload = FunctionTool(
    _kubescape_scan_workload,
    description="""Scan a workload for misconfigurations and vulnerabilities using Kubescape.""",
    name="_kubescape_scan_workload",
)


Scan, ScanConfig = create_typed_fn_tool(kubescape_scan, "kagent.tools.kubescape.Scan", "Scan")
ScanWorkload, ScanWorkloadConfig = create_typed_fn_tool(kubescape_scan_workload, "kagent.tools.kubescape.ScanWorkload", "ScanWorkload")


# Function that runs the istioctl command in the shell
def _run_kubescape_command(command: str) -> str:
    return run_command("kubescape", command.split(" "))
