import { CircularProgress, CircularProgressProps, Typography, Box } from '@mui/material';

function CircularProgressWithLabel(
  props: CircularProgressProps & { value?: number, label: string },
) {
  return (
    <Box sx={{ position: 'relative', display: 'inline-flex' }}>
      <CircularProgress {...props} />
      <Box
        sx={{
          top: 0,
          left: 0,
          bottom: 0,
          right: 0,
          position: 'absolute',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Typography
          variant="caption"
          component="div"
          color={'color' in props ? props['color'] + '.main' : 'text.secondary'}
        >{props.label}</Typography>
      </Box>
    </Box>
  );
}

export default CircularProgressWithLabel