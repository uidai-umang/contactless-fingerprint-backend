ALTER TABLE captures DROP CONSTRAINT captures_finger_type_check;
ALTER TABLE captures ADD CONSTRAINT captures_finger_type_check CHECK (finger_type IN (
    'LEFT_THUMB', 'LEFT_INDEX', 'LEFT_MIDDLE', 'LEFT_RING', 'LEFT_LITTLE',
    'RIGHT_THUMB', 'RIGHT_INDEX', 'RIGHT_MIDDLE', 'RIGHT_RING', 'RIGHT_LITTLE',
    'LEFT_SLAP', 'RIGHT_SLAP'
));