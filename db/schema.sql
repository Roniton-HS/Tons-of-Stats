-- Table for daily stats.
CREATE TABLE
  IF NOT EXISTS
  today (
    id            STRING NOT NULL PRIMARY KEY,
    classic       INT,
    quote         INT,
    ability       INT,
    ability_check BOOL,
    emoji         INT,
    splash        INT,
    splash_check  BOOL,
    elo_change    INT
  );

-- Table for cumulative stats.
CREATE TABLE
  IF NOT EXISTS
  total (
    id            STRING NOT NULL PRIMARY KEY,
    classic       INT,
    quote         INT,
    ability       INT,
    ability_check INT,
    emoji         INT,
    splash        INT,
    splash_check  INT,
    days_played   INT,
    elo           INT
  );
