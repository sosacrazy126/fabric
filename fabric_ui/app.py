import sys, pathlib
pkg_dir = pathlib.Path(__file__).resolve().parent
repo_root = pkg_dir.parent
if str(repo_root) not in sys.path:
    sys.path.insert(0, str(repo_root))

import streamlit as st
import logging
from pathlib import Path

from fabric_ui.config.settings import APP_NAME
from fabric_ui.config.logging_config import setup_logging
from fabric_ui.core.fabric_client import FabricClient
from fabric_ui.core.pattern_manager import PatternManager
from fabric_ui.ui.layout.header import render_header
from fabric_ui.ui.layout.sidebar import render_sidebar
from fabric_ui.ui.views.prompt_hub import PromptHubView

class FabricStudioApp:
    """Main Streamlit controller for Fabric UI."""

    def __init__(self):
        setup_logging()
        self.app_name = APP_NAME
        self.fabric_client = FabricClient()
        self.pattern_manager = PatternManager(self.fabric_client)
        self.views = {
            "Prompt Hub": lambda: PromptHubView(self.fabric_client, self.pattern_manager).render(),
            "Pattern Management": lambda: st.info("Pattern Management coming soon."),
            "Analysis": lambda: st.info("Analysis view coming soon."),
            "Settings": lambda: st.info("Settings coming soon."),
        }

    def inject_custom_css(self):
        css_path = Path(__file__).parent / "static" / "styles.css"
        if css_path.exists():
            with open(css_path) as f:
                st.markdown(f"<style>{f.read()}</style>", unsafe_allow_html=True)

    def main(self):
        st.set_page_config(
            page_title=self.app_name,
            page_icon="🧵",
            layout="wide"
        )
        self.inject_custom_css()
        render_header()
        status = self.fabric_client.get_status()
        view = render_sidebar(status)
        if view in self.views:
            self.views[view]()
        else:
            st.error("Unknown view selected.")

def main():
    FabricStudioApp().main()