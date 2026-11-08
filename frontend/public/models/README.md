# 3D Tux asset

`tux.glb` is the hero model, exported from the supplied Blender project
(`Linux mascot Tux.blend`) and loaded with Three.js `GLTFLoader` in
`src/components/tux/Tux3D.jsx`.

## Verified model facts

Inspected with Blender 5.2.3 (headless) and by reading the GLB JSON chunk:

| Property | Value |
| --- | --- |
| Scene objects | 1 mesh `Tux-printable` (+ a camera in the .blend, not exported) |
| Materials | `black` (#282828), `white` (#ECECEC), `orange` (#E77E13) |
| Textures | none (materials are flat Principled BSDF colours) |
| Rig | none — no armature, no skin |
| Shape keys | none |
| Animation clips | none |
| Triangulated mesh | 41,184 triangles (subdivision level 1 applied at export) |
| Native size | 0.137 × 0.102 × 0.188 units, feet resting near the origin |
| Orientation | Blender Z-up, front facing -Y → exported Y-up, front facing +Z |
| File size | 1.4 MB |

Because the asset is a single rigid mesh, head tracking, eye tracking and
blinking are **not supported** by the model and are intentionally not faked in
the renderer. Only whole-model motion is applied: idle float, cursor lean and
scroll parallax.

`Tux3D` scales the loaded scene to a 1-unit working height, centres it on X/Z
and rests its feet on y = 0. The GLB itself is never modified at runtime.

## Re-exporting

Blender 5.2.3 · `bpy.ops.export_scene.gltf` with:

```
export_format='GLB'
use_selection=True          # only Tux-printable
export_apply=True           # applies the Subdivision Surface modifier
export_yup=True
export_animations=False
export_cameras=False
export_lights=False
export_materials='EXPORT'
export_normals=True
export_texcoords=True
```

The source materials each contained several Material Output nodes, and the
surviving outputs of `white`/`orange` targeted **Cycles only**. The glTF
exporter resolves surfaces through the EEVEE/ALL target, so those materials were
written with no `pbrMetallicRoughness` block and rendered as white metal. Before
exporting, each material must keep exactly one output node whose `target` is
`ALL`, wired from its Principled BSDF — the colours above are the values that
were authored in the file.

## Earlier export

`Linux mascot Tux.glb` is an earlier export of the same mesh, supplied alongside
the Blender file. It carries the broken materials described above (no PBR data)
and is left untouched for reference; `tux.glb` supersedes it.
