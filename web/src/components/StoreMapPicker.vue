<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png'
import markerIcon from 'leaflet/dist/images/marker-icon.png'
import markerShadow from 'leaflet/dist/images/marker-shadow.png'

// Fix default marker icons under Vite
const DefaultIcon = L.icon({
  iconUrl: markerIcon,
  iconRetinaUrl: markerIcon2x,
  shadowUrl: markerShadow,
  iconSize: [25, 41],
  iconAnchor: [12, 41],
  popupAnchor: [1, -34],
  shadowSize: [41, 41],
})
L.Marker.prototype.options.icon = DefaultIcon

const longitude = defineModel<number>('longitude', { default: 0 })
const latitude = defineModel<number>('latitude', { default: 0 })

const mapEl = ref<HTMLElement>()
let map: L.Map | null = null
let marker: L.Marker | null = null

const defaultCenter: L.LatLngExpression = [31.2304, 121.4737] // Shanghai

function hasPoint() {
  return Number(latitude.value) !== 0 || Number(longitude.value) !== 0
}

function syncMarker(lat: number, lng: number, pan = true) {
  if (!map) return
  const pos: L.LatLngExpression = [lat, lng]
  if (!marker) {
    marker = L.marker(pos, { draggable: true }).addTo(map)
    marker.on('dragend', () => {
      const p = marker!.getLatLng()
      latitude.value = Number(p.lat.toFixed(7))
      longitude.value = Number(p.lng.toFixed(7))
    })
  } else {
    marker.setLatLng(pos)
  }
  if (pan) {
    map.setView(pos, Math.max(map.getZoom(), 15))
  }
}

function clearMarker() {
  if (marker && map) {
    map.removeLayer(marker)
    marker = null
  }
  latitude.value = 0
  longitude.value = 0
}

function initMap() {
  if (!mapEl.value || map) return
  const center = hasPoint()
    ? ([latitude.value, longitude.value] as L.LatLngExpression)
    : defaultCenter
  map = L.map(mapEl.value, { center, zoom: hasPoint() ? 16 : 12 })
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; OpenStreetMap',
    maxZoom: 19,
  }).addTo(map)
  map.on('click', (e: L.LeafletMouseEvent) => {
    latitude.value = Number(e.latlng.lat.toFixed(7))
    longitude.value = Number(e.latlng.lng.toFixed(7))
    syncMarker(latitude.value, longitude.value, false)
  })
  if (hasPoint()) {
    syncMarker(latitude.value, longitude.value, false)
  }
  setTimeout(() => map?.invalidateSize(), 80)
}

function applyCoords() {
  if (!hasPoint()) return
  syncMarker(Number(latitude.value), Number(longitude.value), true)
}

watch([longitude, latitude], () => {
  if (!map) return
  if (!hasPoint()) {
    if (marker) {
      map.removeLayer(marker)
      marker = null
    }
    return
  }
  syncMarker(Number(latitude.value), Number(longitude.value), false)
})

onMounted(async () => {
  await nextTick()
  initMap()
})

onBeforeUnmount(() => {
  map?.remove()
  map = null
  marker = null
})

defineExpose({ invalidate: () => map?.invalidateSize() })
</script>

<template>
  <div class="store-map">
    <div class="map-toolbar">
      <el-input-number v-model="longitude" :precision="7" :step="0.0001" controls-position="right" placeholder="经度" />
      <el-input-number v-model="latitude" :precision="7" :step="0.0001" controls-position="right" placeholder="纬度" />
      <el-button @click="applyCoords">定位到坐标</el-button>
      <el-button @click="clearMarker">清除标注</el-button>
    </div>
    <div ref="mapEl" class="map-canvas" />
    <div class="map-hint">点击地图放置门店标注，可拖动标记微调位置</div>
  </div>
</template>

<style scoped>
.store-map { width: 100%; }
.map-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
  align-items: center;
}
.map-canvas {
  width: 100%;
  height: 280px;
  border-radius: 8px;
  border: 1px solid #ebeef5;
  z-index: 0;
}
.map-hint {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
}
</style>
