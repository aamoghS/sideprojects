import pandas as pd
import numpy as np
import re

def extract_title(name):
    if pd.isna(name):
        return "Unknown"
    title_search = re.search(' ([A-Za-z]+)\.', name)
    if title_search:
        return title_search.group(1)
    return "Unknown"

def preprocess_features(df: pd.DataFrame, is_training=False):
    """
    Best practice: Ensure identical preprocessing for training and inference.
    """
    df = df.copy()

    # Feature Engineering: Extract Title
    if 'name' in df.columns:
        df['title'] = df['name'].apply(extract_title)
        # Group rare titles
        df['title'] = df['title'].replace(['Lady', 'Countess','Capt', 'Col','Don', 'Dr', 'Major', 'Rev', 'Sir', 'Jonkheer', 'Dona'], 'Rare')
        df['title'] = df['title'].replace('Mlle', 'Miss')
        df['title'] = df['title'].replace('Ms', 'Miss')
        df['title'] = df['title'].replace('Mme', 'Mrs')
    else:
        df['title'] = 'Unknown'

    # Feature Engineering: Family Size
    if 'sibsp' in df.columns and 'parch' in df.columns:
        df['family_size'] = df['sibsp'] + df['parch'] + 1
        df['is_alone'] = (df['family_size'] == 1).astype(int)
    else:
        df['family_size'] = 1
        df['is_alone'] = 1

    # Fill NaNs logically
    df['age'] = df['age'].fillna(df.groupby('pclass')['age'].transform('median') if 'pclass' in df.columns else df['age'].median())
    df['fare'] = df['fare'].fillna(df['fare'].median())
    
    if 'embarked' in df.columns:
        df['embarked'] = df['embarked'].fillna(df['embarked'].mode()[0])

    # Convert categorical variables natively for LightGBM
    categorical_features = ['sex', 'embarked', 'title']
    for col in categorical_features:
        if col in df.columns:
            df[col] = df[col].astype('category')

    # Drop columns not used for modeling
    drop_cols = ['name', 'ticket', 'cabin', 'boat', 'body', 'home.dest']
    df = df.drop(columns=[c for c in drop_cols if c in df.columns])

    return df
