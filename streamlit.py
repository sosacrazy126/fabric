# filepath: /home/evilbastardxd/Desktop/tools/fabric/streamlit.py

import os
import sys
import json
import time
import shutil
import logging
import pandas as pd
import streamlit as st
from subprocess import run, CalledProcessError
from dotenv import load_dotenv
from typing import Dict, List, Optional, Tuple
from datetime import datetime

# Create formatters
console_formatter = logging.Formatter(
    "\033[92m%(asctime)s\033[0m - "  # Green timestamp
    "\033[94m%(levelname)s\033[0m - "  # Blue level
    "\033[95m[%(funcName)s]\033[0m "  # Purple function name
    "%(message)s"                     # Regular message
)
file_formatter = logging.Formatter(
    "%(asctime)s - %(levelname)s - [%(funcName)s] %(message)s"
)

# Configure root logger
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

# Clear any existing handlers
logger.handlers = []

# Console Handler
console_handler = logging.StreamHandler(sys.stdout)
console_handler.setFormatter(console_formatter)
console_handler.setLevel(logging.INFO)
logger.addHandler(console_handler)

# File Handler
log_dir = os.path.expanduser("~/.config/fabric/logs")
os.makedirs(log_dir, exist_ok=True)
log_file = os.path.join(log_dir, f"fabric_ui_{datetime.now().strftime('%Y%m%d')}.log")
file_handler = logging.FileHandler(log_file)
file_handler.setFormatter(file_formatter)
file_handler.setLevel(logging.DEBUG)  # More detailed logging in file
logger.addHandler(file_handler)

# Detect operating system
PLATFORM = sys.platform

# Log startup message
logger.info("🚀 Fabric UI Starting Up")
logger.info(f"💾 Log file: {log_file}")
logger.info(f"🖥️ Platform detected: {PLATFORM}")

# Global variables
pattern_dir = os.path.expanduser("~/.config/fabric/patterns")
MAX_RETRIES = 3
RETRY_DELAY = 1  # seconds


def initialize_session_state():
    """
    Initialize necessary session state attributes.
    Loads saved outputs from persistent storage if available.
    """
    logger.info("Initializing session state")
    default_configs = {
        "config_loaded": False,
        "vendors": {},
        "available_models": [],
        "selected_vendor": None,
        "selected_model": None,
        "input_content": "",
        "selected_patterns": [],
        "chat_output": [],
        "current_view": "run",
        "wizard_step": "Basic Info",
        "session_name": "",
        "context_name": "",
        "config": {"vendor": "", "model": "", "context_length": "2048"},
        "cached_models": None,
        "last_model_fetch": 0,
        "active_tab": 0,
        "suggest_query": "",
        "suggest_result_count": "15",
        "last_suggest_query": "",
        "last_result_count": "15",
        "last_pattern_index": 0,
        "auto_run_pattern": False,
        "output_logs": [],
        "starred_outputs": [],
        "starring_output": None,
        "temp_star_name": "",
    }

    for key, value in default_configs.items():
        if key not in st.session_state:
            st.session_state[key] = value

    load_saved_outputs()


def parse_models_output(output: str) -> Dict[str, List[str]]:
    """
    Parse the output of 'fabric --listmodels' commands.
    Returns a dictionary mapping provider -> list of available models.
    """
    logger.debug("Parsing models output")
    providers = {}
    current_provider = None

    lines = output.split("\n")
    for line in lines:
        line = line.strip()
        if not line:
            continue
        if line == "Available models:":
            continue
        if not line.startswith("\t") and not line.startswith("["):
            current_provider = line.strip()
            providers[current_provider] = []
        elif current_provider and (line.startswith("\t") or line.startswith("[")):
            model = line.strip()
            if "[" in model and "]" in model:
                model = model.split("]", 1)[1].strip()
            providers[current_provider].append(model)

    logger.debug(f"Found providers: {list(providers.keys())}")
    return providers


def safe_run_command(command: List[str], retry: bool = True) -> Tuple[bool, str, str]:
    """
    Safely run a command with retries. Returns (success, stdout, stderr).
    """
    cmd_str = " ".join(command)
    logger.info(f"Executing command: {cmd_str}")

    for attempt in range(MAX_RETRIES if retry else 1):
        try:
            logger.debug(f"Attempt {attempt + 1}/{MAX_RETRIES if retry else 1}")
            result = run(command, capture_output=True, text=True)
            if result.returncode == 0:
                logger.debug("Command executed successfully")
                return True, result.stdout, ""
            if attempt == MAX_RETRIES - 1 or not retry:
                logger.error(f"Command failed: {result.stderr}")
                return False, "", result.stderr
        except Exception as e:
            if attempt == MAX_RETRIES - 1 or not retry:
                logger.error(f"Command execution failed: {str(e)}")
                return False, "", str(e)
        logger.debug(f"Retrying in {RETRY_DELAY} seconds...")
        time.sleep(RETRY_DELAY)
    logger.error("Max retries exceeded")
    return False, "", "Max retries exceeded"


def fetch_models_once() -> Dict[str, List[str]]:
    """
    Fetch models once, caching results for 5 minutes.
    Returns a dict of providers -> models.
    """
    logger.info("Fetching models")
    current_time = time.time()
    cache_timeout = 300  # 5 minutes

    if (
        st.session_state.cached_models is not None
        and current_time - st.session_state.last_model_fetch < cache_timeout
    ):
        logger.debug("Using cached models")
        return st.session_state.cached_models

    logger.debug("Cache expired or not available, fetching new models")
    success, stdout, stderr = safe_run_command(["fabric", "--listmodels"])
    if not success:
        logger.error(f"Failed to fetch models: {stderr}")
        st.error(f"Failed to fetch models: {stderr}")
        return {}

    providers = parse_models_output(stdout)
    logger.info(f"Found {len(providers)} providers")
    st.session_state.cached_models = providers
    st.session_state.last_model_fetch = current_time
    return providers


def get_configured_providers() -> Dict[str, List[str]]:
    """Get a list of configured providers using 'fabric --listmodels'."""
    return fetch_models_once()


def update_provider_selection(new_provider: str) -> None:
    """
    Update the selected provider and reset related states if changed.
    """
    logger.info(f"Updating provider selection to: {new_provider}")
    if new_provider != st.session_state.config["vendor"]:
        logger.debug("Provider changed, resetting model selection")
        st.session_state.config["vendor"] = new_provider
        st.session_state.selected_vendor = new_provider
        st.session_state.config["model"] = None
        st.session_state.selected_model = None
        st.session_state.available_models = []
        if "model_select" in st.session_state:
            del st.session_state.model_select


def load_configuration() -> bool:
    """
    Load environment variables and initialize configuration.
    Returns True if successful, False otherwise.
    """
    logger.info("Loading configuration")
    try:
        env_path = os.path.expanduser("~/.config/fabric/.env")
        logger.debug(f"Looking for .env file at: {env_path}")

        if not os.path.exists(env_path):
            logger.error(f"Configuration file not found at {env_path}")
            st.error(f"Configuration file not found at {env_path}")
            return False

        load_dotenv(dotenv_path=env_path)
        logger.debug("Environment variables loaded")

        with st.spinner("Loading providers and models..."):
            providers = get_configured_providers()

        if not providers:
            st.error("No providers configured. Please run 'fabric --setup' first.")
            return False

        default_vendor = os.getenv("DEFAULT_VENDOR")
        default_model = os.getenv("DEFAULT_MODEL")
        context_length = os.getenv("DEFAULT_MODEL_CONTEXT_LENGTH", "2048")
        logger.debug(
            f"Default configuration - Vendor: {default_vendor}, Model: {default_model}"
        )

        if not default_vendor or default_vendor not in providers:
            default_vendor = next(iter(providers))
            default_model = providers[default_vendor][0] if providers[default_vendor] else None
            logger.info(
                f"Using fallback configuration - Vendor: {default_vendor}, Model: {default_model}"
            )

        st.session_state.config = {
            "vendor": default_vendor,
            "model": default_model,
            "context_length": context_length,
        }
        st.session_state.vendors = providers
        st.session_state.config_loaded = True
        logger.info("Configuration loaded successfully")
        return True

    except Exception as e:
        logger.error(f"Configuration error: {str(e)}", exc_info=True)
        st.error(f"Configuration error: {str(e)}")
        return False


def load_models_and_providers() -> None:
    """
    Load models and providers from fabric configuration into the Streamlit sidebar.
    """
    try:
        st.sidebar.header("Model and Provider Selection")
        providers: Dict[str, List[str]] = fetch_models_once()

        if not providers:
            st.sidebar.error("No providers configured")
            return

        current_vendor = st.session_state.config.get("vendor", "")
        available_providers = list(providers.keys())

        try:
            provider_index = (
                available_providers.index(current_vendor)
                if current_vendor in available_providers
                else 0
            )
        except ValueError:
            provider_index = 0
            logger.warning(
                f"Current vendor {current_vendor} not found in available providers"
            )

        selected_provider = st.sidebar.selectbox(
            "Provider",
            available_providers,
            index=provider_index,
            key="provider_select",
            on_change=lambda: update_provider_selection(st.session_state.provider_select),
        )

        if selected_provider != st.session_state.config.get("vendor"):
            update_provider_selection(selected_provider)
        st.sidebar.success(f"Using {selected_provider}")

        available_models = providers.get(selected_provider, [])
        if not available_models:
            st.sidebar.warning(f"No models available for {selected_provider}")
            return

        current_model = st.session_state.config.get("model")
        try:
            model_index = (
                available_models.index(current_model)
                if current_model in available_models
                else 0
            )
        except ValueError:
            model_index = 0
            logger.warning(
                f"Current model {current_model} not found in available models for {selected_provider}"
            )

        model_key = f"model_select_{selected_provider}"
        selected_model = st.sidebar.selectbox(
            "Model", available_models, index=model_index, key=model_key
        )

        if selected_model != st.session_state.config.get("model"):
            logger.debug(f"Updating model selection to: {selected_model}")
            st.session_state.config["model"] = selected_model
            st.session_state.selected_model = selected_model

    except Exception as e:
        logger.error(f"Error loading models and providers: {str(e)}", exc_info=True)
        st.sidebar.error(f"Error loading models and providers: {str(e)}")
        st.session_state.selected_model = None
        st.session_state.config["model"] = None


def get_pattern_metadata(pattern_name: str) -> Optional[str]:
    """
    Get pattern metadata from system.md within the named pattern's folder.
    Returns the content as a string, or None if not found.
    """
    pattern_path = os.path.join(pattern_dir, pattern_name, "system.md")
    if os.path.exists(pattern_path):
        with open(pattern_path, "r") as f:
            return f.read()
    return None


def get_patterns() -> List[str]:
    """
    Return the list of available patterns from the configured pattern directory.
    """
    if not os.path.exists(pattern_dir):
        st.error(f"Pattern directory not found: {pattern_dir}")
        return []
    try:
        return [
            item
            for item in os.listdir(pattern_dir)
            if os.path.isdir(os.path.join(pattern_dir, item))
        ]
    except PermissionError:
        st.error(f"Permission error accessing pattern directory: {pattern_dir}")
        return []
    except Exception as e:
        st.error(f"An unexpected error occurred: {e}")
        return []


def load_pattern_metadata() -> List[Dict]:
    """
    Load pattern metadata from Pattern_Descriptions/pattern_descriptions.json if present.
    Returns a list of metadata dictionaries.
    """
    metadata_path = os.path.join(os.path.dirname(__file__), "Pattern_Descriptions/pattern_descriptions.json")
    try:
        with open(metadata_path, "r") as f:
            data = json.load(f)
        logger.info(f"Successfully loaded pattern metadata from {metadata_path}")
        return data.get("patterns", [])
    except FileNotFoundError:
        logger.warning(f"Pattern metadata file not found: {metadata_path}")
        return []
    except json.JSONDecodeError as e:
        logger.error(f"Error parsing pattern metadata JSON: {e}")
        return []


def load_pattern_extracts() -> Dict[str, str]:
    """
    Load pattern extracts from Pattern_Descriptions/pattern_extracts.json if present.
    Returns a dict mapping patternName -> extract text.
    """
    extracts_path = os.path.join(os.path.dirname(__file__), "Pattern_Descriptions/pattern_extracts.json")
    if not os.path.exists(extracts_path):
        logger.warning(f"Pattern extracts file not found: {extracts_path}")
        return {}

    try:
        with open(extracts_path, "r") as f:
            data = json.load(f)
        logger.info(f"Successfully loaded pattern extracts from {extracts_path}")
        extracts = {}
        for pattern in data.get("patterns", []):
            name = pattern.get("patternName")
            extract = pattern.get("pattern_extract", "")
            if name and extract:
                extracts[name] = extract
        return extracts
    except json.JSONDecodeError as e:
        logger.error(f"Error parsing pattern extracts JSON: {e}")
        return {}
    except Exception as e:
        logger.warning(f"Error loading pattern extracts: {e}")
        return {}


def match_patterns(query: str, metadata: List[Dict], use_extracts: bool = True) -> List[Dict]:
    """
    Match metadata patterns based on the user's query string.
    Returns a sorted list of match dictionaries.
    """
    if not query or not metadata:
        return []

    query = query.lower()
    matches = []
    pattern_extracts = load_pattern_extracts() if use_extracts else {}

    for pattern in metadata:
        score = 0
        pattern_name = pattern.get("patternName", "").lower()
        description = pattern.get("description", "").lower()
        tags = [tag.lower() for tag in pattern.get("tags", [])]
        extract_text = pattern_extracts.get(pattern_name, "").lower() if use_extracts else ""

        if query in pattern_name:
            score += 10
            if query == pattern_name or f" {query} " in f" {pattern_name} ":
                score += 5
        if query in description:
            score += 5
        for tag in tags:
            if query in tag:
                score += 3
                if query == tag:
                    score += 2
        if extract_text and query in extract_text:
            score += 2

        query_words = set(query.split())
        for word in query_words:
            if len(word) > 3:
                if word in pattern_name:
                    score += 2
                if word in description:
                    score += 1
                if any(word in tag for tag in tags):
                    score += 1
                if extract_text and word in extract_text:
                    score += 0.5

        if score > 0:
            matches.append({"pattern": pattern, "score": score})

    matches.sort(key=lambda x: x["score"], reverse=True)
    return [m["pattern"] for m in matches]


def create_pattern(pattern_name: str, content: Optional[str] = None) -> Tuple[bool, str]:
    """
    Create a new pattern directory and add an optional system.md file with content.
    Returns (success, message).
    """
    new_pattern_path = None
    try:
        if not pattern_name:
            logger.error("Pattern name cannot be empty")
            return False, "Pattern name cannot be empty."

        new_pattern_path = os.path.join(pattern_dir, pattern_name)
        if os.path.exists(new_pattern_path):
            logger.error(f"Pattern {pattern_name} already exists")
            return False, "Pattern already exists."

        os.makedirs(new_pattern_path)
        logger.info(f"Created pattern directory: {new_pattern_path}")

        if content:
            logger.info(f"Structuring content for pattern '{pattern_name}' via Fabric")
            try:
                current_provider = st.session_state.config.get("vendor")
                current_model = st.session_state.config.get("model")

                if not current_provider or not current_model:
                    raise ValueError("Please select a provider and model first.")

                cmd = ["fabric", "--pattern", "create_pattern"]
                if current_provider and current_model:
                    cmd.extend(["--vendor", current_provider, "--model", current_model])

                logger.debug(f"Running command: {' '.join(cmd)}")
                logger.debug(f"Input content:\n{content}")

                result = run(cmd, input=content, capture_output=True, text=True, check=True)
                structured_content = result.stdout.strip()

                if not structured_content:
                    raise ValueError("No output received from create_pattern")

                system_file = os.path.join(new_pattern_path, "system.md")
                with open(system_file, "w") as f:
                    f.write(structured_content)

                is_valid, validation_message = validate_pattern(pattern_name)
                if not is_valid:
                    raise ValueError(f"Pattern validation failed: {validation_message}")

                logger.info(f"Successfully created pattern '{pattern_name}' with content")

            except CalledProcessError as e:
                logger.error(f"Error running create_pattern: {e.stderr}")
                if os.path.exists(new_pattern_path):
                    shutil.rmtree(new_pattern_path)
                return False, f"Error running create_pattern: {e.stderr}"
            except Exception as e:
                logger.error(f"Unexpected error during content structuring: {str(e)}")
                if os.path.exists(new_pattern_path):
                    shutil.rmtree(new_pattern_path)
                return False, f"Unexpected error: {str(e)}"
        else:
            logger.info(f"Creating minimal template for pattern '{pattern_name}'")
            system_file = os.path.join(new_pattern_path, "system.md")
            with open(system_file, "w") as f:
                f.write("# IDENTITY and PURPOSE\n\n# STEPS\n\n# OUTPUT INSTRUCTIONS\n")

            is_valid, validation_message = validate_pattern(pattern_name)
            if not is_valid:
                logger.warning(f"Pattern created but validation failed: {validation_message}")

        return True, f"Pattern '{pattern_name}' created successfully."

    except Exception as e:
        logger.error(f"Error creating pattern: {str(e)}")
        if new_pattern_path and os.path.exists(new_pattern_path):
            shutil.rmtree(new_pattern_path)
        return False, f"Error creating pattern: {str(e)}"


def delete_pattern(pattern_name: str) -> Tuple[bool, str]:
    """
    Delete an existing pattern by removing its folder.
    Returns (success, message).
    """
    try:
        if not pattern_name:
            return False, "Pattern name cannot be empty."
        pattern_path = os.path.join(pattern_dir, pattern_name)
        if not os.path.exists(pattern_path):
            return False, "Pattern does not exist."
        shutil.rmtree(pattern_path)
        return True, f"Pattern '{pattern_name}' deleted successfully."
    except Exception as e:
        return False, f"Error deleting pattern: {str(e)}"


def pattern_creation_wizard():
    """
    Multi-step wizard for creating a new pattern.
    Gathers inputs in sections, then creates system.md from user inputs.
    """
    st.header("Create New Pattern")
    pattern_name = st.text_input("Pattern Name")
    if pattern_name:
        edit_mode = st.radio("Edit Mode", ["Simple Editor", "Advanced (Wizard)"],
                             key="pattern_creation_edit_mode", horizontal=True)
        if edit_mode == "Simple Editor":
            new_content = st.text_area("Enter Pattern Content", height=400)
            if st.button("Create Pattern", type="primary"):
                success, message = create_pattern(pattern_name, new_content)
                if success:
                    st.success(message)
                    st.experimental_rerun()
                else:
                    st.error(message)
        else:
            sections = ["IDENTITY", "GOAL", "OUTPUT", "OUTPUT INSTRUCTIONS"]
            current_section = st.radio("Edit Section", sections,
                                       key="pattern_creation_section_select")
            if current_section == "IDENTITY":
                identity = st.text_area("Define the IDENTITY", height=200)
                st.session_state.new_pattern_identity = identity
            elif current_section == "GOAL":
                goal = st.text_area("Define the GOAL", height=200)
                st.session_state.new_pattern_goal = goal
            elif current_section == "OUTPUT":
                output = st.text_area("Define the OUTPUT", height=200)
                st.session_state.new_pattern_output = output
            elif current_section == "OUTPUT INSTRUCTIONS":
                instructions = st.text_area("Define the OUTPUT INSTRUCTIONS", height=200)
                st.session_state.new_pattern_instructions = instructions

            pattern_content = f"""# IDENTITY
{st.session_state.get('new_pattern_identity', '')}

# GOAL
{st.session_state.get('new_pattern_goal', '')}

# OUTPUT
{st.session_state.get('new_pattern_output', '')}

# OUTPUT INSTRUCTIONS
{st.session_state.get('new_pattern_instructions', '')}"""

            if st.button("Create Pattern", type="primary"):
                success, message = create_pattern(pattern_name, pattern_content)
                if success:
                    st.success(message)
                    for key in [
                        "new_pattern_identity",
                        "new_pattern_goal",
                        "new_pattern_output",
                        "new_pattern_instructions",
                    ]:
                        if key in st.session_state:
                            del st.session_state[key]
                    st.experimental_rerun()
                else:
                    st.error(message)
    else:
        st.info("Enter a pattern name to create a new pattern")


def bulk_edit_patterns(patterns_to_edit: List[str], field_to_update: str, new_value: str):
    """
    Perform bulk edits on multiple patterns.
    Example usage: editing the PURPOSE in system.md for each pattern.
    """
    results = []
    for pattern in patterns_to_edit:
        try:
            pattern_path = os.path.join(pattern_dir, pattern)
            system_file = os.path.join(pattern_path, "system.md")
            if not os.path.exists(system_file):
                results.append((pattern, False, "system.md not found"))
                continue
            with open(system_file, "r") as f:
                content = f.read()
            if field_to_update == "purpose":
                sections = content.split("#")
                updated_sections = []
                for section in sections:
                    if section.strip().startswith("IDENTITY and PURPOSE"):
                        lines = section.split("\n")
                        for i, line in enumerate(lines):
                            if "You are an AI assistant designed to" in line:
                                lines[i] = f"You are an AI assistant designed to {new_value}."
                        updated_sections.append("\n".join(lines))
                    else:
                        updated_sections.append(section)
                new_content = "#".join(updated_sections)
                with open(system_file, "w") as f:
                    f.write(new_content)
                results.append((pattern, True, "Updated successfully"))
            else:
                results.append(
                    (pattern, False, f"Field {field_to_update} is not supported.")
                    )
        except Exception as e:
            results.append((pattern, False, str(e)))
    return results


def pattern_creation_ui():
    """
    UI component for creating patterns quickly in a single text area.
    """
    pattern_name = st.text_area("Pattern Name")
    if not pattern_name:
        st.info("Enter a pattern name to create a new pattern")
        return

    system_content = """# IDENTITY and PURPOSE

You are an AI assistant designed to {purpose}.

# STEPS

- Step 1
- Step 2
- Step 3

# OUTPUT INSTRUCTIONS

- Output format instructions here
"""
    new_content = st.text_area("Edit Pattern Content", system_content, height=400)
    if st.button("Create Pattern", type="primary"):
        if not pattern_name:
            st.error("Pattern name cannot be empty.")
        else:
            success, message = create_pattern(pattern_name)
            if success:
                system_file = os.path.join(pattern_dir, pattern_name, "system.md")
                with open(system_file, "w") as f:
                    f.write(new_content)
                st.success(f"Pattern '{pattern_name}' created successfully!")
                st.experimental_rerun()
            else:
                st.error(message)


def pattern_management_ui():
    """Placeholder UI component for pattern management in the sidebar."""
    st.sidebar.title("Pattern Management")


def save_output_log(pattern_name: str, input_content: str, output_content: str, timestamp: str):
    """
    Save pattern execution log entry to session state, then persist them.
    """
    log_entry = {
        "timestamp": timestamp,
        "pattern_name": pattern_name,
        "input": input_content,
        "output": output_content,
        "is_starred": False,
        "custom_name": "",
    }
    st.session_state.output_logs.append(log_entry)
    save_outputs()


def star_output(log_index: int, custom_name: str = "") -> bool:
    """
    Star/favorite an output log (by index) and give it an optional custom name.
    Returns True if successfully starred.
    """
    try:
        if 0 <= log_index < len(st.session_state.output_logs):
            log_entry = st.session_state.output_logs[log_index].copy()
            log_entry["is_starred"] = True
            log_entry["custom_name"] = (
                custom_name or f"Starred Output #{len(st.session_state.starred_outputs) + 1}"
            )
            if not any(s["timestamp"] == log_entry["timestamp"] for s in st.session_state.starred_outputs):
                st.session_state.starred_outputs.append(log_entry)
                save_outputs()
                return True
        return False
    except Exception as e:
        logger.error(f"Error starring output: {str(e)}")
        return False


def unstar_output(log_index: int):
    """
    Unstar/remove from favorites an output log by index.
    """
    if 0 <= log_index < len(st.session_state.starred_outputs):
        st.session_state.starred_outputs.pop(log_index)
        save_outputs()


def validate_input_content(input_text: str) -> Tuple[bool, str]:
    """
    Validate input text for minimum length, max size, special char ratio, etc.
    Returns (is_valid, error_message).
    """
    if not input_text or input_text.isspace():
        return False, "Input content cannot be empty."
    if len(input_text.strip()) < 2:
        return False, "Input content must be at least 2 characters long."
    if len(input_text.encode("utf-8")) > 100 * 1024:
        return False, "Input content exceeds maximum size of 100KB."
    special_chars = set("!@#$%^&*()_+[]{}|\\;:'\",.<>?`~")
    special_char_count = sum(1 for c in input_text if c in special_chars)
    special_char_ratio = special_char_count / len(input_text)
    if special_char_ratio > 0.3:
        return False, "Input contains too many special characters."
    control_chars = set(chr(i) for i in range(32) if i not in [9, 10, 13])
    if any(c in control_chars for c in input_text):
        return False, "Input contains invalid control characters."
    try:
        input_text.encode("utf-8").decode("utf-8")
    except UnicodeError:
        return False, "Input contains invalid Unicode characters."
    return True, ""


def sanitize_input_content(input_text: str) -> str:
    """
    Sanitize input text by removing nulls and control chars (except newline, tab, etc.).
    Returns sanitized text.
    """
    text = input_text.replace("\0", "")
    allowed_chars = {"\n", "\t", "\r"}
    sanitized_chars = []
    for c in text:
        if c in allowed_chars or ord(c) >= 32:
            sanitized_chars.append(c)
        else:
            sanitized_chars.append(" ")
    text = "".join(sanitized_chars)
    text = " ".join(text.split())
    return text


def execute_patterns(patterns_to_run: List[str],
    chain_mode: bool = False,
                     initial_input: Optional[str] = None) -> List[str]:
    """
    Execute the selected patterns, optionally chaining output from one into the next.
    Returns a list of all pattern outputs as markdown strings.
    """
    logger.info(f"Executing {len(patterns_to_run)} patterns")
    st.session_state.chat_output = []
    all_outputs = []
    current_input = initial_input or st.session_state.input_content
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    current_provider = st.session_state.config.get("vendor")
    current_model = st.session_state.config.get("model")
    if not current_provider or not current_model:
        error_msg = "Please select a provider and model first."
        logger.error(error_msg)
        st.error(error_msg)
        return all_outputs

    is_valid, error_message = validate_input_content(current_input)
    if not is_valid:
        logger.error(f"Input validation failed: {error_message}")
        st.error(f"Input validation failed: {error_message}")
        return all_outputs
    try:
        sanitized_input = sanitize_input_content(current_input)
        if sanitized_input != current_input:
            logger.info("Input content was sanitized")
            st.warning("Input content was automatically sanitized.")
        current_input = sanitized_input
    except Exception as e:
        logger.error(f"Error sanitizing input: {str(e)}")
        st.error(f"Error processing input: {str(e)}")
        return all_outputs

    execution_info = f"**Using Model:** {current_provider} - {current_model}"
    all_outputs.append(execution_info)
    logger.info(f"Using model: {current_model} from provider: {current_provider}")

    try:
        for pattern in patterns_to_run:
            logger.info(f"Running pattern: {pattern}")
            try:
                cmd = ["fabric", "--pattern", pattern]
                logger.debug(f"Executing command: {' '.join(cmd)}")
                message = current_input if chain_mode else st.session_state.input_content
                input_data = str(message)
                result = run(cmd, input=input_data, capture_output=True, text=True, check=True)
                pattern_output = result.stdout.strip()
                logger.debug(f"Raw output from pattern {pattern}:\n{pattern_output}")

                if pattern_output:
                    output_msg = f"""### {pattern}\n\n{pattern_output}"""
                    all_outputs.append(output_msg)
                    save_output_log(pattern, message, pattern_output, timestamp)
                    if chain_mode:
                        current_input = pattern_output
                else:
                    warning_msg = f"### {pattern}\n\nNo output generated."
                    logger.warning(f"Pattern {pattern} generated no output")
                    all_outputs.append(warning_msg)
            except UnicodeEncodeError as e:
                error_msg = f"### {pattern}\n\n❌ Invalid characters: {str(e)}"
                logger.error(f"Unicode encoding error for pattern {pattern}: {str(e)}")
                all_outputs.append(error_msg)
                if chain_mode:
                    break
            except CalledProcessError as e:
                error_msg = f"### {pattern}\n\n❌ Error executing: {e.stderr.strip()}"
                logger.error(f"Pattern {pattern} failed: {e.stderr.strip()}")
                all_outputs.append(error_msg)
                if chain_mode:
                    break
            except Exception as e:
                error_msg = f"### {pattern}\n\n❌ Failed to execute: {str(e)}"
                logger.error(f"Pattern {pattern} failed: {str(e)}", exc_info=True)
                all_outputs.append(error_msg)
                if chain_mode:
                    break
    except Exception as e:
        error_msg = f"### Error\n\n❌ Error in pattern execution: {str(e)}"
        logger.error(error_msg, exc_info=True)
        st.error(error_msg)

    logger.info("Pattern execution completed")
    return all_outputs


def validate_pattern(pattern_name: str) -> Tuple[bool, str]:
    """
    Validate a pattern's structure for required files/sections.
    Returns (is_valid, message).
    """
    try:
        pattern_path = os.path.join(pattern_dir, pattern_name)
        system_md = os.path.join(pattern_path, "system.md")
        if not os.path.exists(system_md):
            return False, "Missing required file: system.md."
        with open(system_md) as f:
            content = f.read()
            required_sections = ["# IDENTITY", "# STEPS", "# OUTPUT"]
            missing_sections = []
            for section in required_sections:
                if section.lower() not in content.lower():
                    missing_sections.append(section)
            if missing_sections:
                return True, f"Warning: Missing sections: {', '.join(missing_sections)}"
        return True, "Pattern is valid."
    except Exception as e:
        return False, f"Error validating pattern: {str(e)}"


def pattern_editor(pattern_name: str):
    """
    Provide UI to edit an existing pattern's system.md and user.md files.
    Supports both simple text and wizard-based editing.
    """
    if not pattern_name:
        return
    pattern_path = os.path.join(pattern_dir, pattern_name)
    system_file = os.path.join(pattern_path, "system.md")
    user_file = os.path.join(pattern_path, "user.md")

    st.markdown(f"### Editing Pattern: {pattern_name}")
    is_valid, message = validate_pattern(pattern_name)
    if not is_valid:
        st.error(message)
    elif message != "Pattern is valid.":
        st.warning(message)
    else:
        st.success("Pattern structure is valid")

    edit_mode = st.radio("Edit Mode", ["Simple Editor", "Advanced (Wizard)"],
                         key=f"edit_mode_{pattern_name}", horizontal=True)
    if edit_mode == "Simple Editor":
        if os.path.exists(system_file):
            with open(system_file) as f:
                content = f.read()
            new_content = st.text_area("Edit system.md", content, height=600)
            if st.button("Save system.md"):
                with open(system_file, "w") as f:
                    f.write(new_content)
                st.success("Saved successfully!")
        else:
            st.error("system.md file not found")

        if os.path.exists(user_file):
            with open(user_file) as f:
                content = f.read()
            new_content = st.text_area("Edit user.md", content, height=300)
            if st.button("Save user.md"):
                with open(user_file, "w") as f:
                    f.write(new_content)
                st.success("Saved successfully!")
    else:
        if os.path.exists(system_file):
            with open(system_file) as f:
                content = f.read()
            sections = content.split("#")
            edited_sections = []
            for section in sections:
                if not section.strip():
                    continue
                lines = section.strip().split("\n", 1)
                if len(lines) > 1:
                    title, body = lines
                else:
                    title, body = lines[0], ""
                st.markdown(f"#### {title}")
                new_content = st.text_area(f"Edit {title} section",
                                           value=body.strip(), height=200,
                                           key=f"section_{title}")
                edited_sections.append(f"# {title}\n\n{new_content}")
            if st.button("Save Changes"):
                new_content = "\n\n".join(edited_sections)
                with open(system_file, "w") as f:
                    f.write(new_content)
                st.success("Changes saved successfully!")
                is_valid, message = validate_pattern(pattern_name)
                if not is_valid:
                    st.error(message)
                elif message != "Pattern is valid.":
                    st.warning(message)
        else:
            st.error("system.md file not found")


def get_outputs_dir() -> str:
    """
    Get or create the directory where output logs are saved.
    """
    outputs_dir = os.path.expanduser("~/.config/fabric/outputs")
    os.makedirs(outputs_dir, exist_ok=True)
    return outputs_dir


def save_outputs():
    """
    Persist output logs and starred outputs to JSON files in ~/.config/fabric/outputs.
    """
    logger.info("Saving outputs")
    outputs_dir = get_outputs_dir()
    output_logs_file = os.path.join(outputs_dir, "output_logs.json")
    starred_outputs_file = os.path.join(outputs_dir, "starred_outputs.json")

    try:
        with open(output_logs_file, "w") as f:
            json.dump(st.session_state.output_logs, f, indent=2)
        logger.debug(f"Saved output logs to {output_logs_file}")

        with open(starred_outputs_file, "w") as f:
            json.dump(st.session_state.starred_outputs, f, indent=2)
        logger.debug(f"Saved starred outputs to {starred_outputs_file}")
    except PermissionError as e:
        msg = f"Permission denied when saving outputs: {str(e)}"
        logger.error(msg)
        st.error(msg)
    except json.JSONDecodeError as e:
        msg = f"Error encoding outputs to JSON: {str(e)}"
        logger.error(msg)
        st.error(msg)
    except Exception as e:
        msg = f"Unexpected error saving outputs: {str(e)}"
        logger.error(msg)
        st.error(msg)


def load_saved_outputs():
    """
    Load previously saved output logs and starred outputs into session state.
    """
    logger.info("Loading saved outputs")
    outputs_dir = get_outputs_dir()
    output_logs_file = os.path.join(outputs_dir, "output_logs.json")
    starred_outputs_file = os.path.join(outputs_dir, "starred_outputs.json")

    try:
        if os.path.exists(output_logs_file):
            with open(output_logs_file, "r") as f:
                st.session_state.output_logs = json.load(f)
            logger.debug(f"Loaded output logs from {output_logs_file}")

        if os.path.exists(starred_outputs_file):
            with open(starred_outputs_file, "r") as f:
                st.session_state.starred_outputs = json.load(f)
            logger.debug(f"Loaded starred outputs from {starred_outputs_file}")
    except json.JSONDecodeError as e:
        msg = f"Error decoding saved outputs: {str(e)}"
        logger.error(msg)
        st.error(msg)
        st.session_state.output_logs = []
        st.session_state.starred_outputs = []
    except PermissionError as e:
        msg = f"Permission denied when loading outputs: {str(e)}"
        logger.error(msg)
        st.error(msg)
    except Exception as e:
        msg = f"Unexpected error loading saved outputs: {str(e)}"
        logger.error(msg)
        st.error(msg)
        st.session_state.output_logs = []
        st.session_state.starred_outputs = []


def handle_star_name_input(log_index: int, name: str):
    """
    Handle starring an output (with a custom name) during UI interactions.
    """
    try:
        if star_output(log_index, name):
            st.success("Output starred successfully!")
        else:
            st.error("Failed to star output. Please try again.")
    except Exception as e:
        logger.error(f"Error handling star name input: {str(e)}")
        st.error(f"Error starring output: {str(e)}")


def execute_pattern_chain(patterns_sequence: List[str], initial_input: str) -> Dict:
    """
    Execute a sequence of patterns in a chain, passing each pattern's output
    as input to the next. Returns a dict with results and metadata.
    """
    logger.info(f"Starting pattern chain execution with {len(patterns_sequence)} patterns")
    chain_results = {
        "sequence": patterns_sequence,
        "stages": [],
        "final_output": None,
        "metadata": {
            "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "success": False,
        },
    }
    current_input = initial_input

    try:
        for i, pattern in enumerate(patterns_sequence, 1):
            logger.info(f"Chain Stage {i}: {pattern}")
            stage_result = {
                "pattern": pattern,
                "input": current_input,
                "output": None,
                "success": False,
                "error": None,
            }
            try:
                cmd = ["fabric", "--pattern", pattern]
                result = run(cmd, input=current_input, capture_output=True, text=True, check=True)
                output = result.stdout.strip()
                if output:
                    stage_result["output"] = output
                    stage_result["success"] = True
                    current_input = output
                else:
                    stage_result["error"] = "Pattern generated no output"
                    logger.warning(f"Pattern {pattern} generated no output")
            except CalledProcessError as e:
                stage_result["error"] = f"Error executing pattern: {e.stderr.strip()}"
                logger.error(stage_result["error"])
                break
            except Exception as e:
                stage_result["error"] = f"Unexpected error: {str(e)}"
                logger.error(stage_result["error"])
                break
            chain_results["stages"].append(stage_result)
            save_output_log(pattern, stage_result["input"],
                stage_result["output"] or stage_result["error"],
                            chain_results["metadata"]["timestamp"])
        successful_stages = [s for s in chain_results["stages"] if s["success"]]
        if successful_stages:
            chain_results["final_output"] = successful_stages[-1]["output"]
            chain_results["metadata"]["success"] = True
    except Exception as e:
        logger.error(f"Chain execution failed: {str(e)}", exc_info=True)
        chain_results["metadata"]["error"] = str(e)

    return chain_results


def enhance_input_preview():
    """
    Display a preview of the input content with basic statistics (char/word count).
    """
    if "input_content" in st.session_state and st.session_state.input_content:
        with st.expander("Input Preview", expanded=True):
            st.markdown("### Current Input")
            st.code(st.session_state.input_content, language="text")
            char_count = len(st.session_state.input_content)
            word_count = len(st.session_state.input_content.split())
            col1, col2 = st.columns(2)
            with col1:
                st.metric("Characters", char_count)
            with col2:
                st.metric("Words", word_count)


def get_clipboard_content() -> Tuple[bool, str, str]:
    """
    Cross-platform clipboard 'paste' with error handling.
    Returns (success, content, error_message).
    """
    try:
        if PLATFORM == "darwin":
            result = run(["pbpaste"], capture_output=True, text=True, check=True)
            content = result.stdout
        elif PLATFORM == "win32":
            try:
                import pyperclip
                content = pyperclip.paste()
            except ImportError:
                return False, "", "pyperclip is required on Windows. Install with: pip install pyperclip"
            except Exception as e:
                return False, "", f"Windows clipboard error: {str(e)}"
        else:
            result = run(["xclip", "-selection", "clipboard", "-o"],
                         capture_output=True, text=True, check=True)
        content = result.stdout

        # Validate UTF-8
        content.encode("utf-8").decode("utf-8")
        return True, content, ""
    except FileNotFoundError:
        if PLATFORM == "darwin":
            return False, "", "Could not access clipboard on macOS."
        elif PLATFORM == "win32":
            return False, "", "Clipboard access failed. Try installing pyperclip."
        return False, "", "xclip not installed. Install with: sudo apt-get install xclip"
    except CalledProcessError as e:
        return False, "", f"Failed to read clipboard: {e.stderr}"
    except Exception as e:
        return False, "", f"Unexpected error reading clipboard: {str(e)}"


def set_clipboard_content(content: str) -> Tuple[bool, str]:
    """
    Cross-platform clipboard 'copy' with error handling. Returns (success, error_message).
    """
    try:
        try:
            input_bytes = content.encode("utf-8")
        except UnicodeError:
            return False, "Content contains invalid Unicode characters"

        if PLATFORM == "darwin":
            run(["pbcopy"], input=input_bytes, check=True)
        elif PLATFORM == "win32":
            try:
                import pyperclip
                pyperclip.copy(content)
            except ImportError:
                return False, "pyperclip is required on Windows. Install with: pip install pyperclip"
            except Exception as e:
                return False, f"Windows clipboard error: {str(e)}"
        else:
            run(["xclip", "-selection", "clipboard"], input=input_bytes, check=True)
        return True, ""
    except FileNotFoundError:
        if PLATFORM == "darwin":
            return False, "Could not access clipboard on macOS."
        elif PLATFORM == "win32":
            return False, "Clipboard access failed. Try installing pyperclip."
        return False, "xclip not installed. Install with: sudo apt-get install xclip"
    except CalledProcessError as e:
        return False, f"Failed to copy to clipboard: {e.stderr}"
    except Exception as e:
        return False, f"Unexpected error copying to clipboard: {str(e)}"


def add_pattern_to_selection(pattern_name: str):
    """
    Add a pattern to the run list, giving Streamlit feedback via toast.
    """
    sel = st.session_state.selected_patterns
    if pattern_name not in sel:
        sel.append(pattern_name)
        st.toast(f"Added {pattern_name} to selection", icon="✅")
    else:
        st.toast(f"{pattern_name} is already selected", icon="ℹ️")


def main():
    """
    Main Streamlit logic for the Fabric Pattern Studio interface.
    """
    logger.info("Starting Fabric Pattern Studio")
    try:
        st.set_page_config(
            page_title="Fabric Pattern Studio",
            page_icon="🧬",
            layout="wide",
            initial_sidebar_state="expanded",
        )

        st.markdown(
            """
            <style>
                .fabric-header {
                    padding: 1rem;
                    margin-bottom: 1rem;
                    background: linear-gradient(90deg, rgba(155, 108, 255, 0.1) 0%, rgba(76, 181, 255, 0.1) 100%);
                    border-radius: 8px;
                }
                .fabric-title {
                    font-size: 2.5em;
                    margin: 0;
                    background: linear-gradient(90deg, #9B6CFF 0%, #4CB5FF 100%);
                    -webkit-background-clip: text;
                    -webkit-text-fill-color: transparent;
                    font-weight: 600;
                    text-align: center;
                }
                .section-header {
                    font-size: 1.3em;
                    font-weight: 600;
                    margin-top: 1.5em;
                    margin-bottom: 0.5em;
                    color: #4CB5FF;
                }
                .help-text {
                    color: #888;
                    font-size: 0.95em;
                }
                .plugin-area {
                    border: 1px dashed #9B6CFF;
                    padding: 1em;
                    margin: 1em 0;
                    border-radius: 8px;
                    background: rgba(155, 108, 255, 0.03);
                }
            </style>
        """,
            unsafe_allow_html=True,
        )

        st.markdown(
            '<div class="fabric-header"><h1 class="fabric-title">Pattern Studio</h1></div>',
            unsafe_allow_html=True,
        )

        initialize_session_state()

        if "pattern_metadata" not in st.session_state:
            st.session_state.pattern_metadata = load_pattern_metadata()
            st.session_state.pattern_extracts = load_pattern_extracts()

        with st.sidebar:
            st.title("Navigation")
            nav = st.radio("Go to:",
                           ["Prompt Hub", "Pattern Management", "Analysis", "Settings"],
                           key="main_nav_selector")
            st.markdown("---")
            st.title("Configuration")
            load_models_and_providers()
            st.markdown("---")
            st.markdown('<div class="plugin-area">Plugin/Extension Area (future extensibility)</div>',
                        unsafe_allow_html=True)
            st.markdown("---")
            st.markdown(
                """
                <div style='text-align: center; margin-bottom: 1rem;'>
                    <a href="https://github.com/danielmiessler/fabric" target="_blank">
                        <img src="https://img.shields.io/github/stars/danielmiessler/fabric?style=social" alt="GitHub Repo">
                    </a>
                </div>
                """,
                unsafe_allow_html=True,
            )

        if nav == "Prompt Hub":
            st.markdown('<div class="section-header">Prompt Hub 🧩</div>', unsafe_allow_html=True)
            st.markdown(
                '<div class="help-text">Create, run, edit, and organize your prompts. '
                'Star your favorites. Chain patterns. Analyze outputs.</div>',
                unsafe_allow_html=True
            )

            active_tab_index = st.session_state.get("active_tab", 0)
            tab_names = ["Run", "Create", "Edit", "⭐ Starred"]
            tabs = st.tabs(tab_names)
            st.session_state.active_tab = active_tab_index

            with tabs[0]:
                st.markdown('<div class="section-header">Run Patterns</div>', unsafe_allow_html=True)
                patterns = get_patterns()
                if not patterns:
                    st.warning("No patterns available. Create a pattern first.")
                else:
                    st.multiselect(
                        "Select Patterns to Run",
                        patterns,
                        default=st.session_state.selected_patterns,
                        key="selected_patterns",
                        help="Choose one or more patterns to run."
                    )
                    st.markdown("### ✨ Pattern Suggestions")
                    st.markdown(
                        '<div class="help-text">Describe your task and get pattern suggestions.</div>',
                        unsafe_allow_html=True
                    )
                    query = st.text_area(
                        "Describe your task...",
                        key="suggest_query_inline",
                        placeholder="Example: summarize git diff into bullet points",
                        height=100
                    )
                    suggest_col1, suggest_col2 = st.columns([3, 1])
                    with suggest_col1:
                        suggest_button = st.button(
                            "🔍 Suggest Patterns",
                            key="suggest_inline",
                            type="primary",
                            use_container_width=True
                        )
                    with suggest_col2:
                        st.selectbox(
                            "Number of suggestions:",
                            options=["15", "50", "100", "All"],
                            index=0,
                            key="suggest_result_count_inline",
                            help="How many pattern suggestions to show"
                        )
                    if suggest_button and query:
                        with st.spinner("Finding matching patterns..."):
                            metadata = st.session_state.pattern_metadata
                            matches = match_patterns(query, metadata, use_extracts=True)
                            if not matches:
                                st.info("No matching patterns found.")
                            else:
                                total_matches = len(matches)
                                st.success(f"Found {total_matches} matching patterns")
                                result_count_str = st.session_state.get("suggest_result_count_inline", "15")
                                if result_count_str == "All":
                                    display_count = total_matches
                                else:
                                    try:
                                        display_count = int(result_count_str)
                                    except ValueError:
                                        display_count = 15
                                display_matches = matches[:display_count]
                                for i, match in enumerate(display_matches):
                                    pattern_name = match.get("patternName", "")
                                    description = match.get("description", "No description available")
                                    tags = match.get("tags", [])
                                    with st.expander(f"{pattern_name}", expanded=False):
                                        st.markdown(f"**Description:** {description}")
                                        if tags:
                                            st.write("**Tags:**")
                                            tag_cols = st.columns(min(len(tags), 4))
                                            for j, tag_item in enumerate(tags):
                                                with tag_cols[j % len(tag_cols)]:
                                                    st.write(f"- {tag_item}")
                                        pattern_exists = pattern_name in get_patterns()
                                        if pattern_exists:
                                            st.button(
                                                "➕ Add to Selection",
                                                key=f"add_inline_{pattern_name}",
                                                on_click=add_pattern_to_selection,
                                                args=(pattern_name,),
                                                use_container_width=True,
                                            )
                                        else:
                                            st.warning("Pattern not installed locally.")

                    if st.session_state.selected_patterns:
                        for ptn in st.session_state.selected_patterns:
                            with st.expander(f"📝 {ptn} Details", expanded=False):
                                meta = get_pattern_metadata(ptn)
                                if meta:
                                    st.markdown(meta)
                                else:
                                    st.info("No description available")
                        st.subheader("Input")
                        input_method = st.radio("Input Method", ["Clipboard", "Manual Input"], horizontal=True)
                        if input_method == "Clipboard":
                            col_load, col_preview = st.columns([2, 1])
                            with col_load:
                                if st.button("📋 Load from Clipboard", use_container_width=True):
                                    success, content, error = get_clipboard_content()
                                    if success:
                                        valid, msg = validate_input_content(content)
                                        if not valid:
                                            st.error(f"Invalid clipboard content: {msg}")
                                        else:
                                            sanitized = sanitize_input_content(content)
                                            st.session_state.input_content = sanitized
                                            st.session_state.show_preview = True
                                            st.success("Content loaded from clipboard!")
                                    else:
                                        st.error(error)
                            with col_preview:
                                if st.button("👁 Toggle Preview", use_container_width=True):
                                    st.session_state.show_preview = not st.session_state.get("show_preview", False)
                        else:
                            st.session_state.input_content = st.text_area(
                                "Enter Input Text",
                                value=st.session_state.get("input_content", ""),
                                height=200
                            )
                        if st.session_state.get("show_preview", False) or input_method == "Manual Input":
                            if st.session_state.get("input_content"):
                                enhance_input_preview()

                        chain_mode = st.checkbox(
                            "Chain Mode",
                            help="Execute patterns in sequence, passing output from one to the next"
                        )
                        if chain_mode and len(st.session_state.selected_patterns) > 1:
                            st.info("Patterns will be executed in the listed order")
                            st.markdown("##### Drag to reorder patterns:")
                            patterns_df = pd.DataFrame({
                                "Pattern": st.session_state.selected_patterns
                            })
                            edited_df = st.data_editor(
                                patterns_df,
                                use_container_width=True,
                                key="pattern_reorder",
                                hide_index=True,
                                column_config={
                                    "Pattern": st.column_config.TextColumn(
                                        "Pattern", help="Drag to reorder patterns"
                                    )
                                },
                            )
                            new_patterns = edited_df["Pattern"].tolist()
                            if new_patterns != st.session_state.selected_patterns:
                                st.session_state.selected_patterns = new_patterns

                        col1, col2 = st.columns([3, 1])
                        with col1:
                            run_button = st.button("🚀 Run Patterns", type="primary",
                                                   use_container_width=True)
                            auto_run = st.session_state.get("auto_run_pattern", False)

                            if run_button or auto_run:
                                if auto_run:
                                    st.session_state.auto_run_pattern = False
                                if not st.session_state.input_content:
                                    st.warning("Please provide input content.")
                                else:
                                    with st.spinner("Running patterns..."):
                                        if chain_mode:
                                            outputs = execute_patterns(
                                                st.session_state.selected_patterns,
                                                chain_mode=True
                                            )
                                        else:
                                            outputs = execute_patterns(
                                                st.session_state.selected_patterns
                                            )
                                        st.session_state.chat_output = outputs
                        if st.session_state.chat_output:
                            st.markdown("---")
                            st.header("Pattern Outputs")
                            for message in st.session_state.chat_output:
                                st.markdown(message)
                            st.markdown("---")
                            col1, col2 = st.columns(2)
                            with col1:
                                if st.button("📋 Copy All Outputs"):
                                    all_outputs = "\n\n".join(st.session_state.chat_output)
                                    success, error = set_clipboard_content(all_outputs)
                                    if success:
                                        st.success("All outputs copied!")
                                    else:
                                        st.error(error)
                            with col2:
                                if st.button("❌ Clear Outputs"):
                                    st.session_state.chat_output = []
                                    st.success("Outputs cleared!")
                    else:
                        st.info("Select one or more patterns to run.")

            with tabs[1]:
                st.markdown('<div class="section-header">Create New Pattern</div>', unsafe_allow_html=True)
                creation_mode = st.radio(
                    "Creation Mode", ["Simple Editor", "Advanced (Wizard)"],
                    key="creation_mode_main", horizontal=True
                )
                if creation_mode == "Simple Editor":
                    pattern_creation_ui()
                else:
                    pattern_creation_wizard()

            with tabs[2]:
                st.markdown('<div class="section-header">Edit Patterns</div>', unsafe_allow_html=True)
                patterns = get_patterns()
                if not patterns:
                    st.warning("No patterns available. Create a pattern first.")
                else:
                    selected_pattern = st.selectbox(
                        "Select Pattern to Edit", [""] + patterns
                    )
                    if selected_pattern:
                        pattern_editor(selected_pattern)

            with tabs[3]:
                st.markdown('<div class="section-header">⭐ Starred Outputs</div>', unsafe_allow_html=True)
                if not st.session_state.starred_outputs:
                    st.info("No starred outputs yet.")
                else:
                    for i, starred in enumerate(st.session_state.starred_outputs):
                        with st.expander(
                            f"⭐ {starred.get('custom_name', f'Starred #{i+1}')} ({starred['timestamp']})",
                            expanded=False,
                        ):
                            col1, col2 = st.columns([3, 1])
                            with col1:
                                st.markdown(
                                    f"### {starred.get('custom_name', f'Starred Output #{i+1}')}")
                            with col2:
                                edit_key = f"edit_name_{i}"
                                if st.button("✏️ Edit Name", key=edit_key):
                                    st.session_state[f"editing_name_{i}"] = True
                            if st.session_state.get(f"editing_name_{i}", False):
                                new_name = st.text_input(
                                    "New name",
                                    value=starred.get("custom_name", ""),
                                    key=f"name_input_{i}"
                                )
                                if st.button("Save Name", key=f"save_name_{i}"):
                                    starred["custom_name"] = new_name
                                    st.session_state.starred_outputs[i] = starred
                                    save_outputs()
                                    st.session_state[f"editing_name_{i}"] = False
                                    st.experimental_rerun()

                            st.markdown("### Pattern")
                            st.code(starred["pattern_name"], language="text")
                            st.markdown("### Input")
                            st.code(starred["input"], language="text")
                            st.markdown("### Output")
                            st.markdown(starred["output"])
                            col1, col2 = st.columns([1, 4])
                            with col1:
                                if st.button("📋 Copy", key=f"copy_starred_{i}"):
                                    success, error = set_clipboard_content(starred["output"])
                                    if success:
                                        st.success("Starred output copied!")
                                    else:
                                        st.error(error)
                            with col2:
                                if st.button("Remove Star", key=f"remove_star_{i}"):
                                    unstar_output(i)
                                    st.success("Starred output removed.")
                                    st.experimental_rerun()
                    if st.button("Clear All Starred"):
                        if st.checkbox("Confirm clearing all starred outputs"):
                            st.session_state.starred_outputs = []
                            save_outputs()
                            st.success("All starred outputs cleared!")
                            st.experimental_rerun()

        elif nav == "Pattern Management":
            st.markdown('<div class="section-header">Pattern Management 🗂️</div>', unsafe_allow_html=True)
            mgmt_tabs = st.tabs(["Create", "Edit", "Delete"])
            with mgmt_tabs[0]:
                st.markdown('<div class="section-header">Create New Pattern</div>', unsafe_allow_html=True)
                creation_mode = st.radio(
                    "Creation Mode",
                    ["Simple Editor", "Advanced (Wizard)"],
                    key="creation_mode_main_mgmt",
                    horizontal=True
                )
                if creation_mode == "Simple Editor":
                    pattern_creation_ui()
                else:
                    pattern_creation_wizard()

            with mgmt_tabs[1]:
                st.markdown('<div class="section-header">Edit Patterns</div>', unsafe_allow_html=True)
                patterns = get_patterns()
                if not patterns:
                    st.warning("No patterns available. Create a pattern first.")
                else:
                    selected_pattern = st.selectbox(
                        "Select Pattern to Edit", [""] + patterns,
                        key="edit_pattern_select_mgmt"
                    )
                    if selected_pattern:
                        pattern_editor(selected_pattern)

            with mgmt_tabs[2]:
                st.markdown('<div class="section-header">Delete Patterns</div>', unsafe_allow_html=True)
                patterns = get_patterns()
                if not patterns:
                    st.warning("No patterns available.")
                else:
                    patterns_to_delete = st.multiselect(
                        "Select Patterns to Delete",
                        patterns,
                        key="delete_patterns_selector_mgmt",
                    )
                    if patterns_to_delete:
                        st.warning(f"You are about to delete {len(patterns_to_delete)} pattern(s):")
                        for pattern in patterns_to_delete:
                            st.write(f"- {pattern}")
                        confirm_delete = st.checkbox(
                            "I understand that this action cannot be undone",
                            key="confirm_delete_checkbox_mgmt"
                        )
                        if st.button("🗑️ Delete Selected Patterns",
                            type="primary",
                            disabled=not confirm_delete,
                            key="delete_selected_patterns_btn_mgmt"):
                            for pattern in patterns_to_delete:
                                success, msg = delete_pattern(pattern)
                                if success:
                                    st.success(msg)
                                else:
                                    st.error(msg)
                            st.experimental_rerun()
                    else:
                        st.info("Select one or more patterns to delete.")

        elif nav == "Analysis":
            st.markdown('<div class="section-header">Analysis 📊</div>', unsafe_allow_html=True)
            analysis_tabs = st.tabs(["Output Analysis", "History"])
            with analysis_tabs[0]:
                st.markdown('<div class="section-header">Output Analysis</div>', unsafe_allow_html=True)
                if st.session_state.chat_output:
                    for i, output in enumerate(reversed(st.session_state.chat_output), 1):
                        st.markdown(f"**Output #{i}:**")
                        st.markdown(output)
                else:
                    st.info("Run some patterns to see output analysis.")
            with analysis_tabs[1]:
                st.markdown('<div class="section-header">Pattern Output History</div>', unsafe_allow_html=True)
                if not st.session_state.output_logs:
                    st.info("No pattern outputs recorded yet.")
                else:
                    for i, log in enumerate(reversed(st.session_state.output_logs)):
                        with st.expander(f"{log['timestamp']} - {log['pattern_name']}"):
                            st.markdown("### Input")
                            st.code(log["input"], language="text")
                            st.markdown("### Output")
                            st.markdown(log["output"])

        elif nav == "Settings":
            st.markdown('<div class="section-header">Settings & Customization ⚙️</div>', unsafe_allow_html=True)
            st.info("User preferences and UI customization coming soon!")

        st.markdown(
            '<a href="https://github.com/sosacrazy126" target="_blank" class="signature">made by zo6</a>',
            unsafe_allow_html=True,
        )
    except Exception as e:
        logger.error("Unexpected error in main function", exc_info=True)
        st.error(f"An unexpected error occurred: {str(e)}")
        st.stop()


if __name__ == "__main__":
    logger.info("Application startup")
    main()
