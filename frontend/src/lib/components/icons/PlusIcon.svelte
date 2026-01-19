<script lang="ts">
	export let size: number = 16;
	export let color: string = 'currentColor';
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 16 16"
	fill="none"
	xmlns="http://www.w3.org/2000/svg"
>
	<path
		d="M8 3.5V12.5M3.5 8H12.5"
		stroke={color}
		stroke-width="1.5"
		stroke-linecap="round"
		stroke-linejoin="round"
	/>
</svg>
