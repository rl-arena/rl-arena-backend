"""
Script to regenerate HTML replay files from existing JSON replays.

This script is useful when the HTML template is updated and you want to
regenerate all existing replay HTML files without re-running matches.

Usage:
    python scripts/regenerate_replay_html.py
    python scripts/regenerate_replay_html.py --replay-id 37c1b7b6-0f4b-4c1b-9dc9-f0496d49e2f0
    python scripts/regenerate_replay_html.py --all
"""

import json
import sys
import argparse
from pathlib import Path

# Add parent directory to path to import rl_arena
sys.path.insert(0, str(Path(__file__).parent.parent))

from rl_arena.utils.replay import replay_to_html


def convert_executor_to_rl_arena_format(executor_data: dict) -> dict:
    """
    Convert executor replay format to rl-arena format.
    
    Executor format has observations as strings, rl-arena format needs them as dicts.
    """
    converted_frames = []
    
    for frame in executor_data.get('frames', []):
        # Parse observations from strings to lists
        observations_dict = {}
        for agent_id, obs in frame.get('observations', {}).items():
            # Handle both string and list formats
            if isinstance(obs, str):
                # Convert string like "0.5 0.5 0.0 0.0 0.5 0.5 0.0 0.0" to list
                observations_dict[agent_id] = [float(x) for x in obs.strip().split()]
            elif isinstance(obs, list):
                # Already a list
                observations_dict[agent_id] = obs
            else:
                raise ValueError(f"Unknown observation format: {type(obs)}")
        
        converted_frame = {
            "step": frame.get('frame_number', 0),
            "state": observations_dict,  # Key change: observations → state
            "actions": frame.get('actions', {}),
            "rewards": frame.get('rewards', {}),
            "done": frame.get('done', False),
        }
        
        if 'info' in frame:
            converted_frame['info'] = frame['info']
        
        converted_frames.append(converted_frame)
    
    return {
        "metadata": executor_data.get('metadata', {}),
        "frames": converted_frames,
        "version": executor_data.get('version', '1.0'),
    }


def regenerate_single_replay(json_path: Path, output_path: Path = None) -> bool:
    """
    Regenerate HTML from a single JSON replay file.
    
    Args:
        json_path: Path to JSON replay file
        output_path: Optional custom output path. If None, uses same name as JSON with .html extension
        
    Returns:
        True if successful, False otherwise
    """
    try:
        # Read JSON replay
        with open(json_path, 'r', encoding='utf-8') as f:
            executor_data = json.load(f)
        
        # Convert executor format to rl-arena format
        replay_data = convert_executor_to_rl_arena_format(executor_data)
        
        # Determine output path
        if output_path is None:
            output_path = json_path.with_suffix('.html')
        
        # Generate HTML
        print(f"Regenerating: {json_path.name} -> {output_path.name}")
        
        # Extract environment name from metadata
        env_name = replay_data.get('metadata', {}).get('environment', 'Pong')
        
        html_content = replay_to_html(
            recording=replay_data,
            env_name=env_name,
            output_path=None  # Don't save yet, we'll do it manually
        )
        
        # Save HTML
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(html_content)
        
        print(f"✅ Successfully generated: {output_path}")
        return True
        
    except Exception as e:
        print(f"❌ Error processing {json_path.name}: {e}")
        import traceback
        traceback.print_exc()
        return False


def regenerate_all_replays(replay_dir: Path) -> tuple[int, int]:
    """
    Regenerate HTML for all JSON replay files in a directory.
    
    Args:
        replay_dir: Directory containing replay files
        
    Returns:
        Tuple of (success_count, total_count)
    """
    json_files = list(replay_dir.glob('*.json'))
    
    if not json_files:
        print(f"No JSON replay files found in {replay_dir}")
        return 0, 0
    
    print(f"Found {len(json_files)} JSON replay files")
    print("=" * 60)
    
    success_count = 0
    for json_path in json_files:
        if regenerate_single_replay(json_path):
            success_count += 1
        print()
    
    return success_count, len(json_files)


def main():
    parser = argparse.ArgumentParser(
        description='Regenerate HTML replay files from JSON replays'
    )
    parser.add_argument(
        '--replay-dir',
        type=str,
        default='storage/replays',
        help='Directory containing replay files (default: storage/replays)'
    )
    parser.add_argument(
        '--replay-id',
        type=str,
        help='Specific replay ID to regenerate (e.g., 37c1b7b6-...)'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Regenerate all replay HTML files'
    )
    
    args = parser.parse_args()
    
    # Resolve replay directory
    replay_dir = Path(args.replay_dir)
    if not replay_dir.is_absolute():
        replay_dir = Path(__file__).parent.parent / replay_dir
    
    if not replay_dir.exists():
        print(f"❌ Replay directory not found: {replay_dir}")
        return 1
    
    print("🎬 Replay HTML Regeneration Tool")
    print("=" * 60)
    print(f"Replay directory: {replay_dir}")
    print()
    
    # Single replay mode
    if args.replay_id:
        json_path = replay_dir / f"{args.replay_id}.json"
        if not json_path.exists():
            print(f"❌ Replay file not found: {json_path}")
            return 1
        
        success = regenerate_single_replay(json_path)
        return 0 if success else 1
    
    # All replays mode
    elif args.all:
        success_count, total_count = regenerate_all_replays(replay_dir)
        print("=" * 60)
        print(f"✅ Successfully regenerated: {success_count}/{total_count} replays")
        return 0 if success_count == total_count else 1
    
    # Default: show help
    else:
        parser.print_help()
        print()
        print("Examples:")
        print("  # Regenerate a specific replay")
        print("  python scripts/regenerate_replay_html.py --replay-id 37c1b7b6-0f4b-4c1b-9dc9-f0496d49e2f0")
        print()
        print("  # Regenerate all replays")
        print("  python scripts/regenerate_replay_html.py --all")
        return 0


if __name__ == '__main__':
    sys.exit(main())
