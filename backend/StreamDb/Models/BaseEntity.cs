using System.ComponentModel.DataAnnotations.Schema;

namespace StreamDb.Models;

public class BaseEntity: IHasTimestamps
{
    [Column("id")]
    public int Id { get; set; }
    
    [DatabaseGenerated(DatabaseGeneratedOption.Identity), Column("created_at")]
    public DateTime CreatedAt { get; set; }
    
    [DatabaseGenerated(DatabaseGeneratedOption.Computed), Column("updated_at")]
    public DateTime UpdatedAt { get; set; }
    
    [Column("deleted_at")]
    public DateTime? DeletedAt { get; set; }
}