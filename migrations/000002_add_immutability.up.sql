-- 1. Create the function that raises the error
CREATE OR REPLACE FUNCTION prevent_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable: UPDATE and DELETE are not allowed.';
END;
$$ LANGUAGE plpgsql;

-- 2. Attach the trigger to your table
CREATE TRIGGER enforce_immutability
    BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_modification();