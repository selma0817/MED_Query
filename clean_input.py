import numpy as np
import pandas as pd
import re
import os

INPUT_FILE_PATH = "/Users/panyiyan/Desktop/all/work/pipelines/fuzzyMed/inputs/RAW_Med_Update_October_2025.xlsx"

output_dir = os.path.dirname(INPUT_FILE_PATH)
OUTPUT_FILE_PATH = os.path.join(output_dir, "all_med_med_output.xlsx")
COLUMN_PREFIX = "med_med"

print(f"Reading data from: {INPUT_FILE_PATH}")
try:
    # 1. Read the old Excel file into a DataFrame
    df = pd.read_excel(INPUT_FILE_PATH)
    new_df = df.filter(like=COLUMN_PREFIX)
    print(f"Original column count: {df.shape[1]}")
    print(f"Filtered column count: {new_df.shape[1]}")

    if new_df.empty:
        print(f"Warning: No columns found starting with '{COLUMN_PREFIX}'. The output file will be empty.")
    new_df.to_excel(OUTPUT_FILE_PATH, index=False)

    print(f"\nSuccessfully created and saved the new Excel file to:")
    print(f"-> {OUTPUT_FILE_PATH}")

except FileNotFoundError:
    print(f"Error: Input file not found at {INPUT_FILE_PATH}. Please check the path.")
except Exception as e:
    print(f"An unexpected error occurred: {e}")
